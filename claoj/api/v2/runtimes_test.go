package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

func createRuntimeLanguage(t *testing.T, key, name string) models.Language {
	t.Helper()
	l := models.Language{Key: key, Name: name, CommonName: name, Ace: key, Pygments: key,
		Extension: key, Description: name + " desc"}
	require.NoError(t, db.DB.Create(&l).Error)
	return l
}

func createRuntimeJudge(t *testing.T, name string, online bool) models.Judge {
	t.Helper()
	j := models.Judge{Name: name, AuthKey: "key-" + name, Created: time.Now(), Online: online}
	require.NoError(t, db.DB.Omit("Problems", "Runtimes").Create(&j).Error)
	return j
}

func addRuntime(t *testing.T, judge models.Judge, lang models.Language, name, version string, priority int) {
	t.Helper()
	rv := models.RuntimeVersion{JudgeID: judge.ID, LanguageID: lang.ID, Name: name, Version: version, Priority: priority}
	require.NoError(t, db.DB.Create(&rv).Error)
}

func TestRuntimeList_NonStaffHidesOfflineRuntimes(t *testing.T) {
	setupAdminLeftoversDB(t)
	py := createRuntimeLanguage(t, "PY3", "Python 3")
	cpp := createRuntimeLanguage(t, "CPP", "C++")
	online := createRuntimeJudge(t, "w1", true)
	offline := createRuntimeJudge(t, "w2", false)
	addRuntime(t, online, py, "Python", "3.10.1", 0)
	addRuntime(t, online, py, "Python", "3.10.0", 0)
	addRuntime(t, offline, cpp, "GCC", "9.3.0", 0)

	var body struct {
		Data []map[string]any `json:"data"`
	}

	// Anonymous: only online-judge languages, v1 language-list filter.
	w := doRequest(authedRouter(0, http.MethodGet, "/runtimes", RuntimeList), http.MethodGet, "/runtimes", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, "PY3", body.Data[0]["key"])
	runtimes := body.Data[0]["runtimes"].([]any)
	require.Len(t, runtimes, 1)
	entry := runtimes[0].(map[string]any)
	require.Equal(t, "Python", entry["name"])
	require.Equal(t, []any{"3.10.0", "3.10.1"}, entry["versions"])

	// Staff sees every language, including the offline judge's runtime.
	staff := staffUser(t, "runtime-staff")
	w = doRequest(authedRouter(staff.ID, http.MethodGet, "/runtimes", RuntimeList), http.MethodGet, "/runtimes", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
}

func TestRuntimeMatrix_CollapsesHostGroupsAndMarksLatest(t *testing.T) {
	setupAdminLeftoversDB(t)
	py := createRuntimeLanguage(t, "PY3", "Python 3")
	cpp := createRuntimeLanguage(t, "CPP", "C++")
	a := createRuntimeJudge(t, "w1.a", true)
	b := createRuntimeJudge(t, "w1.b", true)
	c := createRuntimeJudge(t, "w2", true)
	addRuntime(t, a, py, "Python", "3.10.1", 0)
	addRuntime(t, a, cpp, "GCC", "9.3.0", 0)
	addRuntime(t, b, py, "Python", "3.10.1", 0)
	addRuntime(t, b, cpp, "GCC", "9.3.0", 0)
	addRuntime(t, c, py, "Python", "3.9.0", 0)

	w := doRequest(authedRouter(0, http.MethodGet, "/runtimes/matrix", RuntimeMatrix),
		http.MethodGet, "/runtimes/matrix", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Judges    []string `json:"judges"`
		Languages []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"languages"`
		Cells []struct {
			Judge      string `json:"judge"`
			LanguageID uint   `json:"language_id"`
			Runtimes   []struct {
				Name     string   `json:"name"`
				Versions []string `json:"versions"`
			} `json:"runtimes"`
			IsLatest bool `json:"is_latest"`
		} `json:"cells"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	// w1.a and w1.b report identical runtimes, so v1 collapses them to "w1".
	require.Equal(t, []string{"w1", "w2"}, body.Judges)
	require.Len(t, body.Languages, 2)
	require.Equal(t, "C++", body.Languages[0].Name)
	require.Equal(t, "Python 3", body.Languages[1].Name)

	latest := map[string]bool{}
	for _, cell := range body.Cells {
		latest[cell.Judge+"|"+itoa(cell.LanguageID)] = cell.IsLatest
	}
	require.True(t, latest["w1|"+itoa(py.ID)])
	require.False(t, latest["w2|"+itoa(py.ID)])
	// C++ is only reported by the (collapsed) w1 column.
	require.True(t, latest["w1|"+itoa(cpp.ID)])
	require.NotContains(t, latest, "w2|"+itoa(cpp.ID))
}

func TestCompareVersionLists(t *testing.T) {
	require.Less(t, compareVersionLists([]string{"3.9.0"}, []string{"3.10.0"}), 0)
	require.Greater(t, compareVersionLists([]string{"3.10.1"}, []string{"3.9.9"}), 0)
	require.Less(t, compareVersionLists([]string{"3.10"}, []string{"3.10.1"}), 0)
	require.Zero(t, compareVersionLists([]string{"3.10.1"}, []string{"3.10.1"}))
	// Unparseable segments fall back to 0 instead of failing the page.
	require.Zero(t, compareVersionLists([]string{"dev"}, []string{"0"}))
}
func TestJudgeList_IncludesRuntimes(t *testing.T) {
	setupAdminLeftoversDB(t)
	require.NoError(t, db.DB.Exec("CREATE TABLE IF NOT EXISTS judge_judge_runtimes (id integer primary key, judge_id integer, language_id integer)").Error)
	py := createRuntimeLanguage(t, "PY3", "Python 3")
	j := createRuntimeJudge(t, "w1", true)
	createRuntimeJudge(t, "w2", true)
	require.NoError(t, db.DB.Exec("INSERT INTO judge_judge_runtimes (judge_id, language_id) VALUES (?, ?)", j.ID, py.ID).Error)

	w := doRequest(authedRouter(0, http.MethodGet, "/judges", JudgeList), http.MethodGet, "/judges", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []struct {
			Name     string `json:"name"`
			Runtimes []struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"runtimes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	require.Equal(t, "PY3", body.Data[0].Runtimes[0].Key)
	require.NotNil(t, body.Data[1].Runtimes)
	require.Empty(t, body.Data[1].Runtimes)
}

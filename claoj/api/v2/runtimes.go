package v2

import (
	"net/http"
	"sort"
	"strings"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// ============================================================
// PUBLIC RUNTIME PAGES (v1 /runtimes + /runtimes/matrix)
// ============================================================

type runtimeEntry struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type runtimeLanguage struct {
	ID          uint           `json:"id"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	ShortName   *string        `json:"short_name"`
	Description string         `json:"description"`
	Runtimes    []runtimeEntry `json:"runtimes"`
}

// parseVersionTuple turns "3.10.1" into (3, 10, 1); non-numeric parts count
// as 0 so a malformed judge-reported version never breaks the page.
func parseVersionTuple(v string) []int {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}

// compareVersionLists mirrors Python's lexicographic list comparison of
// numeric version tuples (v1 sorts with packaging.version).
func compareVersionLists(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		x, y := parseVersionTuple(a[i]), parseVersionTuple(b[i])
		for j := 0; j < len(x) && j < len(y); j++ {
			if x[j] != y[j] {
				if x[j] < y[j] {
					return -1
				}
				return 1
			}
		}
		if len(x) != len(y) {
			if len(x) < len(y) {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// onlineLanguageIDs is v1 LanguageList's non-staff filter: only languages a
// connected judge reports runtimes for (judge/views/language.py:16).
func onlineLanguageIDs() []uint {
	var ids []uint
	db.DB.Model(&models.RuntimeVersion{}).
		Joins("JOIN judge_judge ON judge_judge.id = judge_runtimeversion.judge_id").
		Where("judge_judge.online = ?", true).
		Distinct("judge_runtimeversion.language_id").
		Pluck("judge_runtimeversion.language_id", &ids)
	return ids
}

// RuntimeList - GET /api/v2/runtimes
func RuntimeList(c *gin.Context) {
	query := db.DB.Model(&models.Language{})
	if access := auth.GetAccess(c); !access.IsStaff && !access.IsSuperuser {
		query = query.Where("id IN ?", onlineLanguageIDs())
	}
	var languages []models.Language
	if err := query.Order("key").Find(&languages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	ids := make([]uint, len(languages))
	for i, l := range languages {
		ids[i] = l.ID
	}
	var versions []models.RuntimeVersion
	if len(ids) > 0 {
		if err := db.DB.Where("language_id IN ?", ids).Order("priority").Find(&versions).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	// v1 Language.runtime_versions(): group by runtime name, dedupe versions,
	// and drop the empty version string a judge reports when detection failed.
	grouped := make(map[uint]map[string][]string)
	names := make(map[uint][]string)
	for _, rv := range versions {
		if grouped[rv.LanguageID] == nil {
			grouped[rv.LanguageID] = make(map[string][]string)
		}
		if _, ok := grouped[rv.LanguageID][rv.Name]; !ok {
			grouped[rv.LanguageID][rv.Name] = nil
			names[rv.LanguageID] = append(names[rv.LanguageID], rv.Name)
		}
		if rv.Version == "" {
			continue
		}
		seen := false
		for _, v := range grouped[rv.LanguageID][rv.Name] {
			if v == rv.Version {
				seen = true
				break
			}
		}
		if !seen {
			grouped[rv.LanguageID][rv.Name] = append(grouped[rv.LanguageID][rv.Name], rv.Version)
		}
	}

	items := make([]runtimeLanguage, 0, len(languages))
	for _, l := range languages {
		// v1 skips languages with no runtime rows at all.
		if len(names[l.ID]) == 0 {
			continue
		}
		entries := make([]runtimeEntry, 0, len(names[l.ID]))
		for _, name := range names[l.ID] {
			rs := grouped[l.ID][name]
			sort.Slice(rs, func(i, j int) bool {
				return compareVersionLists([]string{rs[i]}, []string{rs[j]}) < 0
			})
			if rs == nil {
				rs = []string{}
			}
			entries = append(entries, runtimeEntry{Name: name, Versions: rs})
		}
		items = append(items, runtimeLanguage{
			ID:          l.ID,
			Key:         l.Key,
			Name:        l.Name,
			ShortName:   l.ShortName,
			Description: l.Description,
			Runtimes:    entries,
		})
	}

	c.JSON(http.StatusOK, apiList(items))
}

type matrixCell struct {
	Judge      string         `json:"judge"`
	LanguageID uint           `json:"language_id"`
	Runtimes   []runtimeEntry `json:"runtimes"`
	IsLatest   bool           `json:"is_latest"`
}

// RuntimeMatrix - GET /api/v2/runtimes/matrix
// Mirrors v1 version_matrix (judge/views/status.py:67): a language × judge
// table of what each online judge reports. Judges on the same host family that
// report identical version sets collapse into one column named after the
// shared prefix.
func RuntimeMatrix(c *gin.Context) {
	var onlineJudges []models.Judge
	if err := db.DB.Where("online = ?", true).Find(&onlineJudges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var versions []models.RuntimeVersion
	if err := db.DB.Where("judge_id IN (SELECT id FROM judge_judge WHERE online = ?)", true).
		Order("priority").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type judgeData map[uint][]models.RuntimeVersion
	byJudge := make(map[uint]judgeData)
	languageSeen := make(map[uint]bool)
	judgeName := make(map[uint]string, len(onlineJudges))
	for _, j := range onlineJudges {
		judgeName[j.ID] = j.Name
	}
	for _, rv := range versions {
		if byJudge[rv.JudgeID] == nil {
			byJudge[rv.JudgeID] = make(judgeData)
		}
		byJudge[rv.JudgeID][rv.LanguageID] = append(byJudge[rv.JudgeID][rv.LanguageID], rv)
		languageSeen[rv.LanguageID] = true
	}

	sameVersions := func(x, y judgeData) bool {
		if len(x) != len(y) {
			return false
		}
		for lang, xv := range x {
			yv, ok := y[lang]
			if !ok || len(xv) != len(yv) {
				return false
			}
			for i := range xv {
				if xv[i].Name != yv[i].Name || xv[i].Version != yv[i].Version {
					return false
				}
			}
		}
		return true
	}

	// Group judges by the hostname before the last dot, like v1's rpartition.
	var groupOrder []string
	groups := map[string][]uint{}
	for _, j := range onlineJudges {
		if byJudge[j.ID] == nil {
			continue
		}
		key := j.Name
		if idx := strings.LastIndex(j.Name, "."); idx > 0 {
			key = j.Name[:idx]
		}
		if _, ok := groups[key]; !ok {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], j.ID)
	}

	// Column name -> judge id whose runtimes it displays.
	columns := map[string]uint{}
	for _, key := range groupOrder {
		ids := groups[key]
		if len(ids) == 1 {
			columns[judgeName[ids[0]]] = ids[0]
			continue
		}
		size := make([]int, len(ids))
		rep := make([]int, len(ids))
		for i := range size {
			size[i] = 1
			rep[i] = i
		}
		for i := range ids {
			if rep[i] != i {
				continue
			}
			for j := range ids {
				if i != j && sameVersions(byJudge[ids[i]], byJudge[ids[j]]) {
					rep[j] = i
					size[i]++
					size[j] = 0
				}
			}
		}
		best := 0
		for i := range ids {
			if size[i] > size[best] {
				best = i
			}
		}
		columns[key] = ids[best]
		for i, id := range ids {
			if rep[i] != best {
				columns[judgeName[id]] = id
			}
		}
	}

	// Latest version list per language across every displayed column.
	latest := map[uint][]string{}
	for _, judgeID := range columns {
		for lang, list := range byJudge[judgeID] {
			versions := make([]string, len(list))
			for i, rv := range list {
				versions[i] = rv.Version
			}
			if compareVersionLists(versions, latest[lang]) > 0 {
				latest[lang] = versions
			}
		}
	}

	langIDs := make([]uint, 0, len(languageSeen))
	for id := range languageSeen {
		langIDs = append(langIDs, id)
	}
	var languages []models.Language
	if len(langIDs) > 0 {
		if err := db.DB.Where("id IN ?", langIDs).Order("name").Find(&languages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	columnNames := make([]string, 0, len(columns))
	for name := range columns {
		columnNames = append(columnNames, name)
	}
	sort.Strings(columnNames)

	cells := make([]matrixCell, 0)
	for _, name := range columnNames {
		judgeID := columns[name]
		for _, lang := range languages {
			list := byJudge[judgeID][lang.ID]
			if len(list) == 0 {
				continue
			}
			entries := make([]runtimeEntry, len(list))
			versions := make([]string, len(list))
			for i, rv := range list {
				versions[i] = rv.Version
				entries[i] = runtimeEntry{Name: rv.Name, Versions: []string{rv.Version}}
			}
			cells = append(cells, matrixCell{
				Judge:      name,
				LanguageID: lang.ID,
				Runtimes:   entries,
				IsLatest:   compareVersionLists(versions, latest[lang.ID]) == 0,
			})
		}
	}

	type languageItem struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	langItems := make([]languageItem, len(languages))
	for i, l := range languages {
		langItems[i] = languageItem{ID: l.ID, Name: l.Name}
	}

	c.JSON(http.StatusOK, gin.H{
		"judges":    columnNames,
		"languages": langItems,
		"cells":     cells,
	})
}
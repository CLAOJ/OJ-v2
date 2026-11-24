package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// seedTaxonomyAdmin creates the superuser the type/group handlers resolve.
func seedTaxonomyAdmin(t *testing.T) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: "taxadmin", IsActive: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return u
}

func createTaxonomyProblem(t *testing.T, code string, groupID uint) models.Problem {
	t.Helper()
	p := models.Problem{Code: code, Name: code, GroupID: groupID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func createTaxonomyContest(t *testing.T, key string) models.Contest {
	t.Helper()
	c := models.Contest{Key: key, Name: key}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	return c
}

func typeProblemIDs(t *testing.T, typeID uint) []uint {
	t.Helper()
	ids := []uint{}
	require.NoError(t, db.DB.Table("judge_problem_types").
		Where("problemtype_id = ?", typeID).
		Order("problem_id ASC").
		Pluck("problem_id", &ids).Error)
	return ids
}

func storedTagContestIDs(t *testing.T, tagID uint) []uint {
	t.Helper()
	ids := []uint{}
	require.NoError(t, db.DB.Table("judge_contest_tags").
		Where("contesttag_id = ?", tagID).
		Order("contest_id ASC").
		Pluck("contest_id", &ids).Error)
	return ids
}

func TestAdminProblemType_IncludedProblemsReplaceSemantics(t *testing.T) {
	setupAdminDetailDB(t)
	admin := seedTaxonomyAdmin(t)
	p1 := createTaxonomyProblem(t, "p1", 0)
	p2 := createTaxonomyProblem(t, "p2", 0)
	p3 := createTaxonomyProblem(t, "p3", 0)
	p4 := createTaxonomyProblem(t, "p4", 0)

	// A second type holding a problem: a DELETE not scoped to this type would
	// wipe its rows too.
	otherType := models.ProblemType{Name: "greedy", FullName: "Greedy"}
	require.NoError(t, db.DB.Create(&otherType).Error)
	require.NoError(t, db.DB.Exec(
		"INSERT INTO judge_problem_types (problem_id, problemtype_id) VALUES (?, ?)", p4.ID, otherType.ID).Error)

	create := authedRouter(admin.ID, http.MethodPost, "/admin/problem-types", AdminProblemTypeCreate)
	w := doRequest(create, http.MethodPost, "/admin/problem-types",
		`{"name":"dp","full_name":"Dynamic Programming","problem_ids":[`+
			uintJSON(p1.ID)+`,`+uintJSON(p2.ID)+`]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var ptype models.ProblemType
	require.NoError(t, db.DB.Where("name = ?", "dp").First(&ptype).Error)
	require.Equal(t, []uint{p1.ID, p2.ID}, typeProblemIDs(t, ptype.ID))

	// List reports the included count.
	list := authedRouter(admin.ID, http.MethodGet, "/admin/problem-types", AdminProblemTypeList)
	w = doRequest(list, http.MethodGet, "/admin/problem-types", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listBody struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listBody))
	require.EqualValues(t, 2, listBody.Total)
	counts := map[any]any{}
	for _, item := range listBody.Data {
		counts[item["name"]] = item["problem_count"]
	}
	require.EqualValues(t, 2, counts["dp"])
	require.EqualValues(t, 1, counts["greedy"])

	// Detail returns the included ids and their labels.
	detail := authedRouter(admin.ID, http.MethodGet, "/admin/problem-type/:id", AdminProblemTypeDetail)
	w = doRequest(detail, http.MethodGet, "/admin/problem-type/"+uintJSON(ptype.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detailBody map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detailBody))
	require.Equal(t, []any{float64(p1.ID), float64(p2.ID)}, detailBody["problem_ids"])
	require.EqualValues(t, 2, detailBody["problem_count"])
	require.Equal(t, "p1", detailBody["problems"].([]any)[0].(map[string]any)["code"])

	// Update replaces: p1 is removed, p3 added.
	update := authedRouter(admin.ID, http.MethodPatch, "/admin/problem-type/:id", AdminProblemTypeUpdate)
	w = doRequest(update, http.MethodPatch, "/admin/problem-type/"+uintJSON(ptype.ID),
		`{"problem_ids":[`+uintJSON(p2.ID)+`,`+uintJSON(p3.ID)+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []uint{p2.ID, p3.ID}, typeProblemIDs(t, ptype.ID))
	require.Equal(t, []uint{p4.ID}, typeProblemIDs(t, otherType.ID), "the replace must stay scoped to one type")

	// An omitted field leaves the rows unchanged.
	w = doRequest(update, http.MethodPatch, "/admin/problem-type/"+uintJSON(ptype.ID), `{"full_name":"DP"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []uint{p2.ID, p3.ID}, typeProblemIDs(t, ptype.ID))

	// An empty array clears the association.
	w = doRequest(update, http.MethodPatch, "/admin/problem-type/"+uintJSON(ptype.ID), `{"problem_ids":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, typeProblemIDs(t, ptype.ID))
	require.Equal(t, []uint{p4.ID}, typeProblemIDs(t, otherType.ID))
}

func TestAdminProblemType_UnknownProblemIDIs400(t *testing.T) {
	setupAdminDetailDB(t)
	admin := seedTaxonomyAdmin(t)
	p1 := createTaxonomyProblem(t, "p1", 0)

	create := authedRouter(admin.ID, http.MethodPost, "/admin/problem-types", AdminProblemTypeCreate)
	w := doRequest(create, http.MethodPost, "/admin/problem-types",
		`{"name":"dp","full_name":"DP","problem_ids":[`+uintJSON(p1.ID)+`,9999]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "unknown problem IDs: [9999]")
	var count int64
	require.NoError(t, db.DB.Model(&models.ProblemType{}).Count(&count).Error)
	require.Zero(t, count, "a rejected create must roll back")

	ptype := models.ProblemType{Name: "graph", FullName: "Graph"}
	require.NoError(t, db.DB.Create(&ptype).Error)
	require.NoError(t, db.DB.Exec(
		"INSERT INTO judge_problem_types (problem_id, problemtype_id) VALUES (?, ?)", p1.ID, ptype.ID).Error)
	update := authedRouter(admin.ID, http.MethodPatch, "/admin/problem-type/:id", AdminProblemTypeUpdate)
	w = doRequest(update, http.MethodPatch, "/admin/problem-type/"+uintJSON(ptype.ID), `{"problem_ids":[9999]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, []uint{p1.ID}, typeProblemIDs(t, ptype.ID), "a rejected update must roll back, not clear the rows")
}

func TestAdminProblemGroup_CountIsReadOnlyAndSearchFilters(t *testing.T) {
	setupAdminDetailDB(t)
	admin := seedTaxonomyAdmin(t)
	group := models.ProblemGroup{Name: "ioi", FullName: "International Olympiad in Informatics"}
	require.NoError(t, db.DB.Create(&group).Error)
	other := models.ProblemGroup{Name: "icpc", FullName: "ICPC"}
	require.NoError(t, db.DB.Create(&other).Error)
	createTaxonomyProblem(t, "p1", group.ID)
	createTaxonomyProblem(t, "p2", group.ID)

	detail := authedRouter(admin.ID, http.MethodGet, "/admin/problem-group/:id", AdminProblemGroupDetail)
	w := doRequest(detail, http.MethodGet, "/admin/problem-group/"+uintJSON(group.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 2, body["problem_count"])
	require.NotContains(t, body, "problem_ids")

	// Search matches name and full_name, and the total follows the filter.
	list := authedRouter(admin.ID, http.MethodGet, "/admin/problem-groups", AdminProblemGroupList)
	w = doRequest(list, http.MethodGet, "/admin/problem-groups?search=olympiad", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listBody struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listBody))
	require.EqualValues(t, 1, listBody.Total)
	require.Equal(t, "ioi", listBody.Data[0]["name"])

	w = doRequest(list, http.MethodGet, "/admin/problem-groups?search=icpc", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listBody))
	require.EqualValues(t, 1, listBody.Total)
	require.Equal(t, "icpc", listBody.Data[0]["name"])

	// There is no mutation path: problem_ids in create/update is ignored.
	update := authedRouter(admin.ID, http.MethodPatch, "/admin/problem-group/:id", AdminProblemGroupUpdate)
	w = doRequest(update, http.MethodPatch, "/admin/problem-group/"+uintJSON(group.ID),
		`{"full_name":"IOI","problem_ids":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doRequest(detail, http.MethodGet, "/admin/problem-group/"+uintJSON(group.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 2, body["problem_count"])
	var linked int64
	require.NoError(t, db.DB.Model(&models.Problem{}).Where("group_id = ?", group.ID).Count(&linked).Error)
	require.EqualValues(t, 2, linked, "group must never detach its problems")
}

func TestAdminProblemType_SearchFiltersList(t *testing.T) {
	setupAdminDetailDB(t)
	admin := seedTaxonomyAdmin(t)
	require.NoError(t, db.DB.Create(&models.ProblemType{Name: "dp", FullName: "Dynamic Programming"}).Error)
	require.NoError(t, db.DB.Create(&models.ProblemType{Name: "graph", FullName: "Graph Theory"}).Error)

	list := authedRouter(admin.ID, http.MethodGet, "/admin/problem-types", AdminProblemTypeList)
	w := doRequest(list, http.MethodGet, "/admin/problem-types?search=graph", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "graph", body.Data[0]["name"])

	w = doRequest(list, http.MethodGet, "/admin/problem-types?search=dynamic", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "dp", body.Data[0]["name"])
}

func TestAdminContestTag_IncludedContestsReplaceSemantics(t *testing.T) {
	setupAdminDetailDB(t)
	c1 := createTaxonomyContest(t, "c1")
	c2 := createTaxonomyContest(t, "c2")
	c3 := createTaxonomyContest(t, "c3")

	create := authedRouter(1, http.MethodPost, "/admin/contest-tags", AdminContestTagCreate)
	w := doRequest(create, http.MethodPost, "/admin/contest-tags",
		`{"name":"monthly","color":"#3b82f6","contest_ids":[`+uintJSON(c1.ID)+`,`+uintJSON(c2.ID)+`]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var tag models.ContestTag
	require.NoError(t, db.DB.Where("name = ?", "monthly").First(&tag).Error)
	require.Equal(t, []uint{c1.ID, c2.ID}, storedTagContestIDs(t, tag.ID))

	// A second tag shares c1: updating the first must not touch its rows.
	w = doRequest(create, http.MethodPost, "/admin/contest-tags",
		`{"name":"rated","color":"#22c55e","contest_ids":[`+uintJSON(c1.ID)+`]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var other models.ContestTag
	require.NoError(t, db.DB.Where("name = ?", "rated").First(&other).Error)

	// List reports the count.
	list := authedRouter(1, http.MethodGet, "/admin/contest-tags", AdminContestTagList)
	w = doRequest(list, http.MethodGet, "/admin/contest-tags", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listBody struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listBody))
	require.Len(t, listBody.Data, 2)
	for _, item := range listBody.Data {
		want := 2.0
		if item["id"] == float64(other.ID) {
			want = 1
		}
		require.EqualValues(t, want, item["contest_count"])
	}

	// Detail returns ids + labels.
	detail := authedRouter(1, http.MethodGet, "/admin/contest-tag/:id", AdminContestTagDetail)
	w = doRequest(detail, http.MethodGet, "/admin/contest-tag/"+uintJSON(tag.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detailBody map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detailBody))
	require.Equal(t, []any{float64(c1.ID), float64(c2.ID)}, detailBody["contest_ids"])
	require.EqualValues(t, 2, detailBody["contest_count"])
	require.Equal(t, "c1", detailBody["contests"].([]any)[0].(map[string]any)["key"])

	// Replace: c1 removed, c3 added, other tag untouched.
	update := authedRouter(1, http.MethodPatch, "/admin/contest-tag/:id", AdminContestTagUpdate)
	w = doRequest(update, http.MethodPatch, "/admin/contest-tag/"+uintJSON(tag.ID),
		`{"contest_ids":[`+uintJSON(c2.ID)+`,`+uintJSON(c3.ID)+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []uint{c2.ID, c3.ID}, storedTagContestIDs(t, tag.ID))
	require.Equal(t, []uint{c1.ID}, storedTagContestIDs(t, other.ID))

	// Omitted field leaves rows unchanged.
	w = doRequest(update, http.MethodPatch, "/admin/contest-tag/"+uintJSON(tag.ID), `{"description":"Monthly"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []uint{c2.ID, c3.ID}, storedTagContestIDs(t, tag.ID))

	// Empty array clears this tag only.
	w = doRequest(update, http.MethodPatch, "/admin/contest-tag/"+uintJSON(tag.ID), `{"contest_ids":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, storedTagContestIDs(t, tag.ID))
	require.Equal(t, []uint{c1.ID}, storedTagContestIDs(t, other.ID))
}

func TestAdminContestTag_UnknownContestIDIs400(t *testing.T) {
	setupAdminDetailDB(t)
	c1 := createTaxonomyContest(t, "c1")

	create := authedRouter(1, http.MethodPost, "/admin/contest-tags", AdminContestTagCreate)
	w := doRequest(create, http.MethodPost, "/admin/contest-tags",
		`{"name":"monthly","color":"#3b82f6","contest_ids":[9999]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "unknown contest ids: [9999]")
	var count int64
	require.NoError(t, db.DB.Model(&models.ContestTag{}).Count(&count).Error)
	require.Zero(t, count, "a rejected create must roll back")

	tag := models.ContestTag{Name: "monthly", Color: "#3b82f6"}
	require.NoError(t, db.DB.Create(&tag).Error)
	update := authedRouter(1, http.MethodPatch, "/admin/contest-tag/:id", AdminContestTagUpdate)
	w = doRequest(update, http.MethodPatch, "/admin/contest-tag/"+uintJSON(tag.ID), `{"contest_ids":[`+uintJSON(c1.ID)+`,9999]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Empty(t, storedTagContestIDs(t, tag.ID), "a rejected update must not leave partial rows")
}

func TestAdminProblemType_DeleteRemovesJoinRows(t *testing.T) {
	setupAdminDetailDB(t)
	admin := seedTaxonomyAdmin(t)
	p1 := createTaxonomyProblem(t, "p1", 0)
	p2 := createTaxonomyProblem(t, "p2", 0)

	create := authedRouter(admin.ID, http.MethodPost, "/admin/problem-types", AdminProblemTypeCreate)
	w := doRequest(create, http.MethodPost, "/admin/problem-types",
		`{"name":"dp","full_name":"DP","problem_ids":[`+uintJSON(p1.ID)+`]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var ptype models.ProblemType
	require.NoError(t, db.DB.Where("name = ?", "dp").First(&ptype).Error)

	other := models.ProblemType{Name: "graph", FullName: "Graph"}
	require.NoError(t, db.DB.Create(&other).Error)
	require.NoError(t, db.DB.Exec(
		"INSERT INTO judge_problem_types (problem_id, problemtype_id) VALUES (?, ?)", p2.ID, other.ID).Error)

	del := authedRouter(admin.ID, http.MethodDelete, "/admin/problem-type/:id", AdminProblemTypeDelete)
	w = doRequest(del, http.MethodDelete, "/admin/problem-type/"+uintJSON(ptype.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, typeProblemIDs(t, ptype.ID), "delete must drop the type's join rows")
	require.Equal(t, []uint{p2.ID}, typeProblemIDs(t, other.ID), "another type's rows must survive")

	var count int64
	require.NoError(t, db.DB.Model(&models.ProblemType{}).Where("id = ?", ptype.ID).Count(&count).Error)
	require.Zero(t, count)
}

func TestAdminContestTag_DeleteRemovesJoinRows(t *testing.T) {
	setupAdminDetailDB(t)
	c1 := createTaxonomyContest(t, "c1")
	c2 := createTaxonomyContest(t, "c2")

	create := authedRouter(1, http.MethodPost, "/admin/contest-tags", AdminContestTagCreate)
	w := doRequest(create, http.MethodPost, "/admin/contest-tags",
		`{"name":"monthly","color":"#3b82f6","contest_ids":[`+uintJSON(c1.ID)+`,`+uintJSON(c2.ID)+`]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var tag models.ContestTag
	require.NoError(t, db.DB.Where("name = ?", "monthly").First(&tag).Error)

	w = doRequest(create, http.MethodPost, "/admin/contest-tags",
		`{"name":"rated","color":"#22c55e","contest_ids":[`+uintJSON(c1.ID)+`]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var other models.ContestTag
	require.NoError(t, db.DB.Where("name = ?", "rated").First(&other).Error)

	del := authedRouter(1, http.MethodDelete, "/admin/contest-tag/:id", AdminContestTagDelete)
	w = doRequest(del, http.MethodDelete, "/admin/contest-tag/"+uintJSON(tag.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, storedTagContestIDs(t, tag.ID), "delete must drop the tag's join rows")
	require.Equal(t, []uint{c1.ID}, storedTagContestIDs(t, other.ID), "another tag's rows must survive")

	var count int64
	require.NoError(t, db.DB.Model(&models.ContestTag{}).Where("id = ?", tag.ID).Count(&count).Error)
	require.Zero(t, count)
}

// uintJSON renders an id for inline JSON bodies.
func uintJSON(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

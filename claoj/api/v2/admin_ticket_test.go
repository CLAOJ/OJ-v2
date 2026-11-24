package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

type ticketFixture struct {
	root, plain  models.AuthUser
	plainProfile models.Profile
	// creator is a profile whose id deliberately differs from its auth_user id,
	// so code that mixes the two id spaces (judge_ticket.user_id is a
	// judge_profile id) fails instead of accidentally matching.
	creator   models.Profile
	problemCT models.DjangoContentType
	contestCT models.DjangoContentType
}

func setupAdminTickets(t *testing.T) ticketFixture {
	t.Helper()
	root, plain, plainProfile := setupSmallGapsDB(t)
	f := ticketFixture{root: root, plain: plain, plainProfile: plainProfile}

	// django_content_type ids depend on migration order. Seed unrelated rows
	// first so problem/contest are not 1/2 (the historic hardcoded values): the
	// linked-item tests then really prove model-name resolution.
	for _, ct := range []models.DjangoContentType{
		{AppLabel: "judge", Model: "submission"},
		{AppLabel: "judge", Model: "contestparticipation"},
	} {
		require.NoError(t, db.DB.Create(&ct).Error)
	}
	f.problemCT = models.DjangoContentType{AppLabel: "judge", Model: "problem"}
	f.contestCT = models.DjangoContentType{AppLabel: "judge", Model: "contest"}
	require.NoError(t, db.DB.Create(&f.problemCT).Error)
	require.NoError(t, db.DB.Create(&f.contestCT).Error)
	require.NotEqual(t, uint(1), f.problemCT.ID)
	require.NotEqual(t, uint(2), f.contestCT.ID)

	// One auth_user without a profile shifts every later profile id, so
	// creator.ID != creator.UserID.
	require.NoError(t, db.DB.Create(&models.AuthUser{Username: "noprofile", IsActive: true}).Error)
	creatorUser := models.AuthUser{Username: "carol", IsActive: true}
	require.NoError(t, db.DB.Create(&creatorUser).Error)
	f.creator = models.Profile{UserID: creatorUser.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.creator).Error)
	require.NotEqual(t, f.creator.ID, f.creator.UserID)
	return f
}

func (f ticketFixture) problem(t *testing.T, code, name string) models.Problem {
	t.Helper()
	p := models.Problem{Code: code, Name: name}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func (f ticketFixture) contest(t *testing.T, key, name string) models.Contest {
	t.Helper()
	c := models.Contest{Key: key, Name: name, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	return c
}

func (f ticketFixture) ticket(t *testing.T, title string, contentTypeID, objectID uint) models.Ticket {
	t.Helper()
	tk := models.Ticket{Title: title, UserID: f.creator.ID, Time: time.Now(), ContentTypeID: contentTypeID, ObjectID: objectID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&tk).Error)
	return tk
}

func (f ticketFixture) rootProfileID(t *testing.T) uint {
	t.Helper()
	var p models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", f.root.ID).First(&p).Error)
	return p.ID
}

// ticketPermRouter mirrors the route-level auth.RequirePerm gate in router.go,
// which authedRouter (used for success paths) bypasses.
func ticketPermRouter(userID uint, method, path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path,
		func(c *gin.Context) { c.Set("user_id", userID) },
		auth.RequirePerm("judge.change_ticket"),
		handler)
	return router
}

func TestAdminTicketDetailLinkedItem(t *testing.T) {
	f := setupAdminTickets(t)
	p := f.problem(t, "p1", "Two Sum")
	c := f.contest(t, "round1", "Round One")
	router := authedRouter(f.root.ID, http.MethodGet, "/admin/ticket/:id", AdminTicketDetail)

	detail := func(tk models.Ticket) map[string]any {
		w := doRequest(router, http.MethodGet, "/admin/ticket/"+itoa(tk.ID), "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body
	}

	t.Run("problem", func(t *testing.T) {
		body := detail(f.ticket(t, "problem ticket", f.problemCT.ID, p.ID))
		require.Equal(t, "problem", body["linked_type"])
		require.Equal(t, float64(p.ID), body["linked_id"])
		require.Equal(t, "p1 – Two Sum", body["linked_label"])
		require.Equal(t, "/problem/p1", body["linked_path"])
	})

	t.Run("contest", func(t *testing.T) {
		body := detail(f.ticket(t, "contest ticket", f.contestCT.ID, c.ID))
		require.Equal(t, "contest", body["linked_type"])
		require.Equal(t, float64(c.ID), body["linked_id"])
		require.Equal(t, "round1 – Round One", body["linked_label"])
		require.Equal(t, "/contest/round1", body["linked_path"])
	})

	t.Run("unlinked and orphaned resolve to empty", func(t *testing.T) {
		for _, tk := range []models.Ticket{
			f.ticket(t, "no link", 0, 0),
			f.ticket(t, "orphan", f.problemCT.ID, 999),
			f.ticket(t, "unknown content type", 9999, 1),
		} {
			body := detail(tk)
			require.Equal(t, "", body["linked_type"])
			require.Equal(t, float64(0), body["linked_id"])
			require.Equal(t, "", body["linked_label"])
			require.Equal(t, "", body["linked_path"])
		}
	})
}

func TestTicketDetailUserSideStillResolvesLinkedItem(t *testing.T) {
	f := setupAdminTickets(t)
	p := f.problem(t, "p1", "Two Sum")
	tk := f.ticket(t, "user ticket", f.problemCT.ID, p.ID)
	tk.UserID = f.plainProfile.ID
	require.NoError(t, db.DB.Save(&tk).Error)

	router := authedRouter(f.plain.ID, http.MethodGet, "/ticket/:id", TicketDetail)
	w := doRequest(router, http.MethodGet, "/ticket/"+itoa(tk.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		LinkedItem *struct {
			Type string `json:"type"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"linked_item"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.LinkedItem)
	require.Equal(t, "problem", body.LinkedItem.Type)
	require.Equal(t, "p1", body.LinkedItem.Code)
	require.Equal(t, "Two Sum", body.LinkedItem.Name)
}

func TestAdminTicketCreatorRelation(t *testing.T) {
	f := setupAdminTickets(t)
	tk := f.ticket(t, "creator check", 0, 0)

	// judge_ticket.user_id holds a judge_profile id. The fixture guarantees it
	// differs from the creator's auth_user id, so an id-space mixup fails; a
	// has-one guess (judge_profile.user_id = ticket.id) also fails, because it
	// resolves a different profile.
	require.NotEqual(t, f.creator.ID, f.creator.UserID)
	require.Equal(t, f.creator.ID, tk.UserID)

	detailRouter := authedRouter(f.root.ID, http.MethodGet, "/admin/ticket/:id", AdminTicketDetail)
	w := doRequest(detailRouter, http.MethodGet, "/admin/ticket/"+itoa(tk.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detail map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	require.Equal(t, "carol", detail["creator"])
	require.Equal(t, float64(f.creator.ID), detail["creator_id"])

	listRouter := authedRouter(f.root.ID, http.MethodGet, "/admin/tickets", AdminTicketList)
	w = doRequest(listRouter, http.MethodGet, "/admin/tickets", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list struct {
		Items []struct {
			ID   uint   `json:"id"`
			User string `json:"user"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)
	require.Equal(t, tk.ID, list.Items[0].ID)
	require.Equal(t, "carol", list.Items[0].User)
}

func TestAdminTicketUpdate(t *testing.T) {
	f := setupAdminTickets(t)
	p := f.problem(t, "p1", "Two Sum")
	c := f.contest(t, "round1", "Round One")
	rootProfileID := f.rootProfileID(t)
	tk := f.ticket(t, "old title", 0, 0)
	router := authedRouter(f.root.ID, http.MethodPatch, "/admin/ticket/:id", AdminTicketUpdate)
	target := "/admin/ticket/" + itoa(tk.ID)

	load := func() models.Ticket {
		var got models.Ticket
		require.NoError(t, db.DB.First(&got, tk.ID).Error)
		return got
	}
	patch := func(body string) int {
		return doRequest(router, http.MethodPatch, target, body).Code
	}

	require.Equal(t, http.StatusOK, patch(`{"title":"new title","user_id":`+itoa(f.plainProfile.ID)+`,"linked_type":"problem","linked_id":`+itoa(p.ID)+`}`))
	got := load()
	require.Equal(t, "new title", got.Title)
	require.Equal(t, f.plainProfile.ID, got.UserID)
	require.Equal(t, f.problemCT.ID, got.ContentTypeID)
	require.Equal(t, p.ID, got.ObjectID)

	// user_id is a judge_profile id, not an auth_user id: creator.ID differs
	// from creator.UserID, so storing the auth id fails here.
	require.Equal(t, http.StatusOK, patch(`{"user_id":`+itoa(f.creator.ID)+`}`))
	got = load()
	require.Equal(t, f.creator.ID, got.UserID)
	require.NotEqual(t, f.creator.UserID, got.UserID)

	require.Equal(t, http.StatusOK, patch(`{"user_id":`+itoa(rootProfileID)+`}`))
	require.Equal(t, rootProfileID, load().UserID)

	require.Equal(t, http.StatusOK, patch(`{"linked_type":"contest","linked_id":`+itoa(c.ID)+`}`))
	got = load()
	require.Equal(t, f.contestCT.ID, got.ContentTypeID)
	require.Equal(t, c.ID, got.ObjectID)

	require.Equal(t, http.StatusOK, patch(`{"linked_type":"","linked_id":0}`))
	got = load()
	require.Zero(t, got.ContentTypeID)
	require.Zero(t, got.ObjectID)

	// Validation failures leave the row unchanged.
	for name, body := range map[string]string{
		"type without id":    `{"linked_type":"problem"}`,
		"id without type":    `{"linked_id":1}`,
		"type with zero id":  `{"linked_type":"problem","linked_id":0}`,
		"empty type with id": `{"linked_type":"","linked_id":5}`,
		"unknown type":       `{"linked_type":"blog","linked_id":1}`,
		"unknown object":     `{"linked_type":"problem","linked_id":999}`,
		"blank title":        `{"title":"   "}`,
		"unknown creator":    `{"user_id":999}`,
	} {
		require.Equal(t, http.StatusBadRequest, patch(body), name)
	}
	got = load()
	require.Equal(t, "new title", got.Title)
	require.Equal(t, rootProfileID, got.UserID)
	require.Zero(t, got.ContentTypeID)

	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodPatch, "/admin/ticket/999", `{"title":"x"}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPatch, "/admin/ticket/abc", `{"title":"x"}`).Code)

	// Creating with a title, then clearing it through the creator field must not
	// be possible: an empty title is rejected above; time stays untouched.
	before := load().Time
	require.Equal(t, http.StatusOK, patch(`{"user_id":`+itoa(rootProfileID)+`}`))
	require.True(t, before.Equal(load().Time))

	// No permission -> 403 through the route gate.
	gated := ticketPermRouter(f.plain.ID, http.MethodPatch, "/admin/ticket/:id", AdminTicketUpdate)
	require.Equal(t, http.StatusForbidden, doRequest(gated, http.MethodPatch, target, `{"title":"hacked"}`).Code)
	require.Equal(t, "new title", load().Title)
}

func TestAdminTicketMessageUpdateDelete(t *testing.T) {
	f := setupAdminTickets(t)
	tk := f.ticket(t, "with messages", 0, 0)
	other := f.ticket(t, "other", 0, 0)
	msg := models.TicketMessage{TicketID: tk.ID, UserID: f.creator.ID, Body: "original", Time: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&msg).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.TicketMessage{
		TicketID: other.ID, UserID: f.creator.ID, Body: "other body", Time: time.Now(),
	}).Error)

	patchRouter := authedRouter(f.root.ID, http.MethodPatch, "/admin/ticket/:id/message/:messageId", AdminTicketMessageUpdate)
	deleteRouter := authedRouter(f.root.ID, http.MethodDelete, "/admin/ticket/:id/message/:messageId", AdminTicketMessageDelete)
	patch := func(ticketID uint, body string) int {
		return doRequest(patchRouter, http.MethodPatch, "/admin/ticket/"+itoa(ticketID)+"/message/"+itoa(msg.ID), body).Code
	}
	load := func() models.TicketMessage {
		var got models.TicketMessage
		require.NoError(t, db.DB.First(&got, msg.ID).Error)
		return got
	}

	require.Equal(t, http.StatusOK, patch(tk.ID, `{"body":"edited"}`))
	require.Equal(t, "edited", load().Body)

	// The author is not editable; an unknown user_id in the body is ignored.
	require.Equal(t, http.StatusOK, patch(tk.ID, `{"body":"edited again","user_id":`+itoa(f.plainProfile.ID)+`}`))
	require.Equal(t, "edited again", load().Body)
	require.Equal(t, f.creator.ID, load().UserID)

	require.Equal(t, http.StatusBadRequest, patch(tk.ID, `{"body":"   "}`))
	require.Equal(t, "edited again", load().Body)

	// A message reached through the wrong ticket is a 404, for edit and delete.
	require.Equal(t, http.StatusNotFound, patch(other.ID, `{"body":"stolen"}`))
	require.Equal(t, http.StatusNotFound, doRequest(deleteRouter, http.MethodDelete, "/admin/ticket/"+itoa(other.ID)+"/message/"+itoa(msg.ID), "").Code)
	require.Equal(t, "edited again", load().Body)

	require.Equal(t, http.StatusNotFound, doRequest(patchRouter, http.MethodPatch, "/admin/ticket/"+itoa(tk.ID)+"/message/999", `{"body":"x"}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(patchRouter, http.MethodPatch, "/admin/ticket/"+itoa(tk.ID)+"/message/abc", `{"body":"x"}`).Code)

	// No permission -> 403 through the route gate.
	gated := ticketPermRouter(f.plain.ID, http.MethodDelete, "/admin/ticket/:id/message/:messageId", AdminTicketMessageDelete)
	require.Equal(t, http.StatusForbidden, doRequest(gated, http.MethodDelete, "/admin/ticket/"+itoa(tk.ID)+"/message/"+itoa(msg.ID), "").Code)

	require.Equal(t, http.StatusOK, doRequest(deleteRouter, http.MethodDelete, "/admin/ticket/"+itoa(tk.ID)+"/message/"+itoa(msg.ID), "").Code)
	var n int64
	require.NoError(t, db.DB.Model(&models.TicketMessage{}).Where("id = ?", msg.ID).Count(&n).Error)
	require.Zero(t, n)
}

func TestAdminTicketListFilteredTotal(t *testing.T) {
	f := setupAdminTickets(t)
	seed := func(title string, open bool) models.Ticket {
		tk := f.ticket(t, title, 0, 0)
		tk.IsOpen = open
		require.NoError(t, db.DB.Save(&tk).Error)
		return tk
	}
	assigned := seed("bug one", true)
	seed("bug two", true)
	seed("bug three", true)
	seed("bug closed", false)
	seed("other", true)
	require.NoError(t, db.DB.Model(&assigned).Association("Assignees").Append(&f.plainProfile))

	router := authedRouter(f.root.ID, http.MethodGet, "/admin/tickets", AdminTicketList)
	get := func(query string) (total int, items int) {
		w := doRequest(router, http.MethodGet, "/admin/tickets?"+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Total, len(body.Items)
	}

	total, items := get("search=bug&status=open")
	require.Equal(t, 3, total)
	require.Equal(t, 3, items)

	total, _ = get("status=open")
	require.Equal(t, 4, total)

	total, items = get("assigned=true")
	require.Equal(t, 1, total, "assignee filter must not duplicate or over-count")
	require.Equal(t, 1, items)

	total, _ = get("assigned=false")
	require.Equal(t, 4, total)

	total, _ = get("status=closed")
	require.Equal(t, 1, total)
}

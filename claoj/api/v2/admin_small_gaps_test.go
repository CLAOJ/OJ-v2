package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupSmallGapsDB(t *testing.T) (root, plain models.AuthUser, plainProfile models.Profile) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ProblemClarification{},
		&models.Ticket{}, &models.TicketMessage{},
	))
	db.DB = database

	root = models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&root).Error)
	plain = models.AuthUser{Username: "bob", IsActive: true}
	require.NoError(t, db.DB.Create(&plain).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: root.ID, Timezone: "UTC"}).Error)
	plainProfile = models.Profile{UserID: plain.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&plainProfile).Error)
	return root, plain, plainProfile
}

func TestProblemClarificationUpdate(t *testing.T) {
	root, plain, _ := setupSmallGapsDB(t)
	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	clar := models.ProblemClarification{ProblemID: p.ID, Description: "old text", Date: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&clar).Error)

	const route = "/admin/problem/clarification/:id"
	target := "/admin/problem/clarification/" + strconv.Itoa(int(clar.ID))
	load := func() models.ProblemClarification {
		var got models.ProblemClarification
		require.NoError(t, db.DB.First(&got, clar.ID).Error)
		return got
	}

	t.Run("non-editor is forbidden", func(t *testing.T) {
		w := doRequest(authedRouter(plain.ID, http.MethodPatch, route, ProblemClarificationUpdate), http.MethodPatch, target, `{"description":"hacked"}`)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		require.Equal(t, "old text", load().Description)
	})

	router := authedRouter(root.ID, http.MethodPatch, route, ProblemClarificationUpdate)

	t.Run("empty description is rejected", func(t *testing.T) {
		w := doRequest(router, http.MethodPatch, target, `{"description":""}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.Equal(t, "old text", load().Description)
	})

	t.Run("bad ids", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPatch, "/admin/problem/clarification/abc", `{"description":"x"}`).Code)
		require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodPatch, "/admin/problem/clarification/999", `{"description":"x"}`).Code)
	})

	t.Run("editor updates", func(t *testing.T) {
		w := doRequest(router, http.MethodPatch, target, `{"description":"new text"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, "new text", load().Description)

		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, "new text", body["description"])
		require.Equal(t, float64(clar.ID), body["id"])
	})
}

func TestAdminTicketDelete(t *testing.T) {
	root, _, plainProfile := setupSmallGapsDB(t)

	seed := func(title string) models.Ticket {
		tk := models.Ticket{Title: title, UserID: plainProfile.ID, Time: time.Now(), ContentTypeID: 1, ObjectID: 1}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&tk).Error)
		for i := 0; i < 2; i++ {
			require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.TicketMessage{
				TicketID: tk.ID, UserID: plainProfile.ID, Body: "hello", Time: time.Now(),
			}).Error)
		}
		require.NoError(t, db.DB.Model(&tk).Association("Assignees").Append(&plainProfile))
		return tk
	}
	doomed := seed("doomed")
	kept := seed("kept")

	count := func(table, where string, arg any) int64 {
		var n int64
		require.NoError(t, db.DB.Table(table).Where(where, arg).Count(&n).Error)
		return n
	}

	router := authedRouter(root.ID, http.MethodDelete, "/admin/ticket/:id", AdminTicketDelete)
	w := doRequest(router, http.MethodDelete, "/admin/ticket/"+strconv.Itoa(int(doomed.ID)), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, int64(0), count("judge_ticket", "id = ?", doomed.ID))
	require.Equal(t, int64(0), count("judge_ticketmessage", "ticket_id = ?", doomed.ID))
	require.Equal(t, int64(0), count("judge_ticket_assignees", "ticket_id = ?", doomed.ID))

	// The other ticket keeps its messages and assignee.
	require.Equal(t, int64(1), count("judge_ticket", "id = ?", kept.ID))
	require.Equal(t, int64(2), count("judge_ticketmessage", "ticket_id = ?", kept.ID))
	require.Equal(t, int64(1), count("judge_ticket_assignees", "ticket_id = ?", kept.ID))

	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/ticket/999", "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/ticket/abc", "").Code)
}

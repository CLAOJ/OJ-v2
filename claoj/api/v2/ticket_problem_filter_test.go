package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func problemTickets(t *testing.T, userID uint, target string) (ids []uint, status int) {
	t.Helper()
	router := authedRouter(userID, http.MethodGet, "/tickets", TicketList)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	if w.Code != http.StatusOK {
		return nil, w.Code
	}
	var body struct {
		Items []struct {
			ID uint `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, it := range body.Items {
		ids = append(ids, it.ID)
	}
	require.Equal(t, len(ids), body.Total)
	return ids, w.Code
}

// v1 /problem/<p>/tickets (ProblemTicketListView): the problem's editors see
// every ticket about it, other users only the ones they opened, and a problem
// they cannot access is a 404 — never a peek at its tickets.
func TestTicketList_ProblemFilterPermissions(t *testing.T) {
	f := setupAdminTickets(t)

	problem := f.problem(t, "p1", "Two Sum")
	require.NoError(t, db.DB.Model(&problem).Update("is_public", true).Error)
	hidden := f.problem(t, "secret", "Hidden")
	require.NoError(t, db.DB.Model(&hidden).Update("is_public", false).Error)

	// bob is the problem's author and holds edit_own_problem after this.
	require.NoError(t, db.DB.Model(&problem).Association("Authors").Append(&f.plainProfile))
	grantSubmissionPerm(t, f.plain.ID, "edit_own_problem")

	theirs := f.ticket(t, "carol's ticket", f.problemCT.ID, problem.ID)
	mine := models.Ticket{Title: "bob's ticket", UserID: f.plainProfile.ID, Time: time.Now(),
		ContentTypeID: f.problemCT.ID, ObjectID: problem.ID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&mine).Error)
	hiddenTicket := f.ticket(t, "on hidden problem", f.problemCT.ID, hidden.ID)

	// Editor: every ticket on the problem, and nothing from another problem.
	ids, code := problemTickets(t, f.plain.ID, "/tickets?problem=p1")
	require.Equal(t, http.StatusOK, code)
	require.ElementsMatch(t, []uint{theirs.ID, mine.ID}, ids)

	// A plain user who owns one ticket sees only that one.
	ids, code = problemTickets(t, f.creator.UserID, "/tickets?problem=p1")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []uint{theirs.ID}, ids)

	// A plain user with no tickets on it gets an empty list, not other people's.
	ids, code = problemTickets(t, f.root.ID, "/tickets?problem=p1")
	require.Equal(t, http.StatusOK, code)
	require.ElementsMatch(t, []uint{theirs.ID, mine.ID}, ids) // root is a superuser: an editor

	// A hidden problem is a 404 for anyone who cannot edit it.
	_, code = problemTickets(t, f.creator.UserID, "/tickets?problem=secret")
	require.Equal(t, http.StatusNotFound, code)
	// The editor still sees it.
	ids, code = problemTickets(t, f.root.ID, "/tickets?problem=secret")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []uint{hiddenTicket.ID}, ids)

	// Unknown problem code: 404.
	_, code = problemTickets(t, f.root.ID, "/tickets?problem=nope")
	require.Equal(t, http.StatusNotFound, code)
}
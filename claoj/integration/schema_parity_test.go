package integration

import (
	"os"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authv2 "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestSchemaParity verifies every GORM model SELECTs cleanly against a real
// deployment database. A real deployment database is:
//
//  1. Django's schema, via:  cd OJ && python manage.py migrate  (against an empty MySQL db)
//  2. PLUS the OJ-v2 additive schema, via: mysql < OJ-v2/scripts/v2_runtime_tables.sql
//
// (2) is required because OJ-v2 retains several v2-only tables/columns on
// top of the Django schema -- see docs/schema-audit.md section 3
// ("RETAINED-V2") for the full list and rationale. Provision both, then run:
//
//	CLAOJ_DJANGO_DB_DSN="user:pass@tcp(127.0.0.1:3306)/claoj_schema?parseTime=true" \
//	  go test ./integration/ -run TestSchemaParity
//
// Skips (does not fail) when CLAOJ_DJANGO_DB_DSN is unset, so `go test ./...`
// stays green without a live MySQL instance.
func TestSchemaParity(t *testing.T) {
	dsn := os.Getenv("CLAOJ_DJANGO_DB_DSN")
	if dsn == "" {
		t.Skip("CLAOJ_DJANGO_DB_DSN not set")
	}
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Django-owned tables (created by `manage.py migrate` alone).
	djangoModels := []interface{}{
		&models.AuthUser{}, &models.AuthGroup{}, &models.AuthPermission{},
		&models.DjangoContentType{}, &models.AuthUserGroup{},
		&models.AuthGroupPermission{}, &models.AuthUserPermission{},
		&models.Profile{}, &models.Organization{}, &models.OrganizationRequest{},
		&models.WebAuthnCredential{},
		&models.Problem{}, &models.ProblemGroup{}, &models.ProblemType{},
		&models.License{}, &models.ProblemTranslation{}, &models.ProblemClarification{},
		&models.LanguageLimit{},
		&models.ProblemData{}, &models.ProblemTestCase{},
		&models.Contest{}, &models.ContestTag{}, &models.ContestAnnouncement{},
		&models.ContestParticipation{},
		&models.ContestProblem{}, &models.ContestSubmission{}, &models.Rating{},
		&models.Submission{}, &models.SubmissionSource{}, &models.SubmissionTestCase{},
		&models.Comment{}, &models.CommentVote{}, &models.CommentLock{},
		&models.GeneralIssue{}, &models.Ticket{}, &models.TicketMessage{},
		&models.BlogPost{}, &models.BlogVote{}, &models.MiscConfig{}, &models.NavigationBar{},
		&models.Language{}, &models.Judge{}, &models.RuntimeVersion{},
		// models.Solution is deliberately NOT in this list: its 4 extra
		// columns (is_official, valid_until, summary, language) are
		// RETAINED-V2 -- see the second loop below.
	}
	for _, m := range djangoModels {
		// Select every mapped column of one row; unknown columns error out.
		err := database.Limit(1).Find(m).Error
		require.NoErrorf(t, err, "model %T does not match Django schema", m)
	}

	// RETAINED-V2 objects: additive v2-only tables, plus a Django-owned
	// table (judge_solution) carrying additive v2-only columns. These only
	// exist once scripts/v2_runtime_tables.sql has also been applied.
	// See docs/schema-audit.md section 3 for the full rationale.
	retainedV2Models := []interface{}{
		&models.Solution{}, // base Django columns + 4 RETAINED-V2 columns
		&models.Notification{}, &models.NotificationPreference{},
		&models.TotpDevice{}, &models.BackupCode{},
		&authv2.OAuthUserLink{},
		&v2.MossResult{},
		&models.CommentRevision{},
		&models.ContestClarification{},
	}
	for _, m := range retainedV2Models {
		err := database.Limit(1).Find(m).Error
		require.NoErrorf(t, err, "RETAINED-V2 model %T does not match deployed schema "+
			"(did you run scripts/v2_runtime_tables.sql?)", m)
	}
}

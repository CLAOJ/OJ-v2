package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupAdminUserCreateDB(t *testing.T) {
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
	))
	db.DB = database
	userService = nil
	require.NoError(t, db.DB.Create(&models.Language{Key: "py3", Name: "Python 3"}).Error)
}

// seedAccount inserts an auth_user + profile and returns both.
func seedAccount(t *testing.T, username string, staff, superuser bool) (models.AuthUser, models.Profile) {
	t.Helper()
	u := models.AuthUser{
		Username: username, Email: username + "@example.com", Password: "old-hash",
		IsActive: true, IsStaff: staff, IsSuperuser: superuser,
	}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return u, p
}

func TestAdminUserCreate_CreatesActiveUserAndProfile(t *testing.T) {
	setupAdminUserCreateDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/users", AdminUserCreate)
	w := doRequest(router, http.MethodPost, "/admin/users",
		`{"username":"newbie","email":"newbie@example.com","password":"Passw0rdOK","full_name":"New Bie"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.Where("username = ?", "newbie").First(&u).Error)
	require.True(t, u.IsActive, "admin-created accounts skip email verification")
	require.Equal(t, "newbie@example.com", u.Email)
	require.Equal(t, "New Bie", u.FirstName)
	require.NotEqual(t, "Passw0rdOK", u.Password)
	ok, err := auth.CheckPassword("Passw0rdOK", u.Password)
	require.NoError(t, err)
	require.True(t, ok)

	var p models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", u.ID).First(&p).Error)
	require.Equal(t, "UTC", p.Timezone)
	require.NotZero(t, p.LanguageID)

	var body struct {
		Success bool `json:"success"`
		User    struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, p.ID, body.User.ID, "the admin API identifies users by profile id")
	require.Equal(t, "newbie", body.User.Username)
}

func TestAdminUserCreate_InactiveWhenAsked(t *testing.T) {
	setupAdminUserCreateDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/users", AdminUserCreate)
	w := doRequest(router, http.MethodPost, "/admin/users",
		`{"username":"sleeper","email":"sleeper@example.com","password":"Passw0rdOK","is_active":false}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.Where("username = ?", "sleeper").First(&u).Error)
	require.False(t, u.IsActive)
}

func TestAdminUserCreate_Rejects(t *testing.T) {
	setupAdminUserCreateDB(t)
	seedAccount(t, "taken", false, false)
	router := authedRouter(1, http.MethodPost, "/admin/users", AdminUserCreate)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"bad username", `{"username":"bad name!","email":"a@example.com","password":"Passw0rdOK"}`, http.StatusBadRequest},
		{"weak password", `{"username":"fine","email":"a@example.com","password":"short"}`, http.StatusBadRequest},
		{"bad email", `{"username":"fine","email":"nope","password":"Passw0rdOK"}`, http.StatusBadRequest},
		{"missing password", `{"username":"fine","email":"a@example.com"}`, http.StatusBadRequest},
		{"duplicate username", `{"username":"taken","email":"other@example.com","password":"Passw0rdOK"}`, http.StatusConflict},
		{"duplicate email", `{"username":"fine","email":"taken@example.com","password":"Passw0rdOK"}`, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(router, http.MethodPost, "/admin/users", tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	var users, profiles int64
	require.NoError(t, db.DB.Model(&models.AuthUser{}).Count(&users).Error)
	require.NoError(t, db.DB.Model(&models.Profile{}).Count(&profiles).Error)
	require.Equal(t, int64(1), users, "a rejected create must not leave an auth_user row")
	require.Equal(t, int64(1), profiles, "a rejected create must not leave a profile row")
}

func TestAdminUserSetPassword_ChangesHash(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/password", AdminUserSetPassword)
	w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(targetProfile.ID))+"/password", `{"password":"N3wPassword"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	ok, err := auth.CheckPassword("N3wPassword", got.Password)
	require.NoError(t, err)
	require.True(t, ok)
}

// Setting a password is an account takeover primitive, so a staff user who is
// not a superuser must not be able to use it on a staff or superuser account.
func TestAdminUserSetPassword_StaffTargetNeedsSuperuser(t *testing.T) {
	setupAdminUserCreateDB(t)
	staff, _ := seedAccount(t, "staff", true, false)
	for _, tc := range []struct {
		name             string
		staff, superuser bool
	}{
		{"otherstaff", true, false},
		{"theroot", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, targetProfile := seedAccount(t, tc.name, tc.staff, tc.superuser)
			router := authedRouter(staff.ID, http.MethodPost, "/admin/user/:id/password", AdminUserSetPassword)
			w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(targetProfile.ID))+"/password", `{"password":"N3wPassword"}`)
			require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

			var got models.AuthUser
			require.NoError(t, db.DB.First(&got, target.ID).Error)
			require.Equal(t, "old-hash", got.Password, "a refused request must not change the password")
		})
	}
}

func TestAdminUserSetPassword_Errors(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "bob", false, false)
	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/password", AdminUserSetPassword)
	id := strconv.Itoa(int(targetProfile.ID))

	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/user/"+id+"/password", `{"password":"weak"}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/user/"+id+"/password", `{}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/user/abc/password", `{"password":"N3wPassword"}`).Code)
	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodPost, "/admin/user/999/password", `{"password":"N3wPassword"}`).Code)
}

// judge_profile has no ban-expiry column, so a ban cannot carry a duration;
// the endpoint must not demand one.
func TestAdminUserBan_DayIsOptional(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "spammer", false, false)

	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/ban", AdminUserBan)
	w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(targetProfile.ID))+"/ban", `{"reason":"spam"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Profile
	require.NoError(t, db.DB.First(&got, targetProfile.ID).Error)
	require.NotNil(t, got.BanReason)
	require.Equal(t, "spam", *got.BanReason)
}

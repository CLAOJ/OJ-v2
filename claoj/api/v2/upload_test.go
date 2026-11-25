package v2

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

// Without R2 the sign endpoints answer 501 (the client then falls back to the
// multipart uploads), and commit only accepts keys under the caller's own
// tmp/<purpose>/<profile>/ prefix, so nobody can claim someone else's upload.
func TestUploadGuards(t *testing.T) {
	fx := setupProblemListDB(t)
	var profile models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", fx.admin.ID).First(&profile).Error)

	sign := authedRouter(fx.admin.ID, http.MethodPost, "/user/avatar/sign", AvatarUploadSign)
	w := doRequest(sign, http.MethodPost, "/user/avatar/sign", `{"content_type":"image/png","size":100}`)
	require.Equal(t, http.StatusNotImplemented, w.Code, w.Body.String())

	set := authedRouter(fx.admin.ID, http.MethodPut, "/user/avatar", AvatarSet)
	for _, key := range []string{
		"tmp/avatar/999999/x.png",                     // another profile
		"tmp/image/" + uintStr(profile.ID) + "/x.png", // another purpose
		"tmp/avatar/" + uintStr(profile.ID) + "/../../avatars/x.png",
		"avatars/x.png",
	} {
		w := doRequest(set, http.MethodPut, "/user/avatar", `{"key":"`+key+`"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, key)
	}
}

func uintStr(v uint) string { return strconv.FormatUint(uint64(v), 10) }

package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, "/", opts.Path)
	assert.Equal(t, GetMaxAge(), opts.MaxAge)
	assert.True(t, opts.Secure)
	assert.True(t, opts.HTTPOnly)
	assert.Equal(t, http.SameSiteStrictMode, opts.SameSite)
}

func TestSet(t *testing.T) {
	w := httptest.NewRecorder()

	opts := Options{
		Name:     "test_cookie",
		Value:    "test_value",
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	Set(w, opts)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "test_cookie", cookies[0].Name)
	assert.Equal(t, "test_value", cookies[0].Value)
	assert.Equal(t, "/", cookies[0].Path)
	assert.Equal(t, 3600, cookies[0].MaxAge)
	assert.True(t, cookies[0].Secure)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
}

func TestSetRefreshToken(t *testing.T) {
	w := httptest.NewRecorder()

	SetRefreshToken(w, "my-refresh-token")

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, RefreshTokenCookieName, cookies[0].Name)
	assert.Equal(t, "my-refresh-token", cookies[0].Value)
	assert.Equal(t, "/", cookies[0].Path)
	assert.Equal(t, GetMaxAge(), cookies[0].MaxAge)
	assert.True(t, cookies[0].Secure)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
}

func TestGet(t *testing.T) {
	t.Run("cookie exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "test_cookie",
			Value: "test_value",
		})

		value, err := Get(req, "test_cookie")

		assert.NoError(t, err)
		assert.Equal(t, "test_value", value)
	})

	t.Run("cookie does not exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		value, err := Get(req, "nonexistent")

		assert.Error(t, err)
		assert.Equal(t, "", value)
		assert.Equal(t, http.ErrNoCookie, err)
	})
}

func TestGetRefreshToken(t *testing.T) {
	t.Run("refresh token exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  RefreshTokenCookieName,
			Value: "my-refresh-token",
		})

		token, err := GetRefreshToken(req)

		assert.NoError(t, err)
		assert.Equal(t, "my-refresh-token", token)
	})

	t.Run("refresh token does not exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		token, err := GetRefreshToken(req)

		assert.Error(t, err)
		assert.Equal(t, "", token)
	})
}

func TestHasCookie(t *testing.T) {
	t.Run("cookie exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "test_cookie",
			Value: "test_value",
		})

		assert.True(t, HasCookie(req, "test_cookie"))
	})

	t.Run("cookie does not exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		assert.False(t, HasCookie(req, "nonexistent"))
	})
}

func TestHasRefreshToken(t *testing.T) {
	t.Run("refresh token exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  RefreshTokenCookieName,
			Value: "my-refresh-token",
		})

		assert.True(t, HasRefreshToken(req))
	})

	t.Run("refresh token does not exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		assert.False(t, HasRefreshToken(req))
	})
}

func TestDelete(t *testing.T) {
	w := httptest.NewRecorder()

	Delete(w, "test_cookie")

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "test_cookie", cookies[0].Name)
	assert.Equal(t, "", cookies[0].Value)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestDeleteRefreshToken(t *testing.T) {
	w := httptest.NewRecorder()

	DeleteRefreshToken(w)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, RefreshTokenCookieName, cookies[0].Name)
	assert.Equal(t, "", cookies[0].Value)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestSetWithExpiry(t *testing.T) {
	w := httptest.NewRecorder()
	expiry := time.Now().Add(24 * time.Hour)

	SetWithExpiry(w, "test_cookie", "test_value", expiry)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "test_cookie", cookies[0].Name)
	assert.Equal(t, "test_value", cookies[0].Value)
	assert.True(t, cookies[0].Secure)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
	// Expires should be set to the provided expiry time (allowing 1 second tolerance)
	assert.WithinDuration(t, expiry, cookies[0].Expires, time.Second)
}

func TestCookieConstants(t *testing.T) {
	assert.Equal(t, "refresh_token", RefreshTokenCookieName)
	// MaxAge is now dynamic based on config (may be 0 in test environment)
	assert.GreaterOrEqual(t, GetMaxAge(), 0)
}

func TestGetMaxAge(t *testing.T) {
	maxAge := GetMaxAge()
	// In test environment, RefreshExpireMin may be 0
	assert.GreaterOrEqual(t, maxAge, 0)
	// If maxAge > 0, it should be divisible by 60 (minutes to seconds conversion)
	if maxAge > 0 {
		assert.Equal(t, 0, maxAge%60)
	}
}

func TestOptionsWithCustomValues(t *testing.T) {
	w := httptest.NewRecorder()

	opts := Options{
		Name:     "custom_cookie",
		Value:    "custom_value",
		Path:     "/api",
		Domain:   "example.com",
		MaxAge:   1800,
		Secure:   false,
		HTTPOnly: false,
		SameSite: http.SameSiteLaxMode,
	}

	Set(w, opts)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "custom_cookie", cookies[0].Name)
	assert.Equal(t, "custom_value", cookies[0].Value)
	assert.Equal(t, "/api", cookies[0].Path)
	assert.Equal(t, "example.com", cookies[0].Domain)
	assert.Equal(t, 1800, cookies[0].MaxAge)
	assert.False(t, cookies[0].Secure)
	assert.False(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
}

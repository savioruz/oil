package cookie

import (
	"net/http"
	"oil/config"
	"oil/shared/constant"
	"time"
)

const (
	RefreshToken = "refresh_token"
)

// GetMaxAge returns the cookie max age in seconds from config
func GetMaxAge() int {
	cfg := config.Get()

	return cfg.JWT.RefreshExpireMin * constant.MinutesToSeconds
}

// Options represents cookie configuration options
type Options struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	MaxAge   int
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

// DefaultOptions returns secure cookie options with sensible defaults
func DefaultOptions() Options {
	return Options{
		Path:     "/",
		MaxAge:   GetMaxAge(),
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

// Set sets a cookie with the provided options
func Set(w http.ResponseWriter, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     opts.Name,
		Value:    opts.Value,
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   opts.MaxAge,
		Secure:   opts.Secure,
		HttpOnly: opts.HTTPOnly, // http.Cookie uses HttpOnly, not HTTPOnly
		SameSite: opts.SameSite,
	})
}

// SetRefreshToken sets a refresh token cookie with secure defaults
func SetRefreshToken(w http.ResponseWriter, token string) {
	opts := DefaultOptions()
	opts.Name = RefreshToken
	opts.Value = token
	Set(w, opts)
}

// Get retrieves a cookie value by name from the request
func Get(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

// GetRefreshToken retrieves the refresh token from the cookie
func GetRefreshToken(r *http.Request) (string, error) {
	return Get(r, RefreshToken)
}

// HasCookie checks if a cookie exists in the request
func HasCookie(r *http.Request, name string) bool {
	_, err := r.Cookie(name)

	return err == nil
}

// HasRefreshToken checks if a refresh token cookie exists
func HasRefreshToken(r *http.Request) bool {
	return HasCookie(r, RefreshToken)
}

// Delete deletes a cookie by setting its MaxAge to -1
func Delete(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:   name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// DeleteRefreshToken deletes the refresh token cookie
func DeleteRefreshToken(w http.ResponseWriter) {
	Delete(w, RefreshToken)
}

// SetWithExpiry sets a cookie with a specific expiration time instead of MaxAge
func SetWithExpiry(w http.ResponseWriter, name, value string, expiry time.Time) {
	opts := DefaultOptions()
	opts.Name = name
	opts.Value = value
	opts.MaxAge = 0 // MaxAge takes precedence, so set to 0 to use Expires

	cookie := &http.Cookie{
		Name:     opts.Name,
		Value:    opts.Value,
		Path:     opts.Path,
		Domain:   opts.Domain,
		Expires:  expiry,
		Secure:   opts.Secure,
		HttpOnly: opts.HTTPOnly, // http.Cookie uses HttpOnly, not HTTPOnly
		SameSite: opts.SameSite,
	}

	http.SetCookie(w, cookie)
}

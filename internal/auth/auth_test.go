package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_NoCookie(t *testing.T) {
	h := &Handler{jwtSecret: []byte("test-secret")}
	var gotClaims *Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = ClaimsFromCtx(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/problems", nil)
	h.Middleware(next).ServeHTTP(httptest.NewRecorder(), req)

	assert.Nil(t, gotClaims)
}

func TestMiddleware_ValidCookie(t *testing.T) {
	secret := []byte("test-secret")
	h := &Handler{jwtSecret: secret}

	claims := &Claims{
		Sub:      "user123",
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "github",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	require.NoError(t, err)

	var gotClaims *Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = ClaimsFromCtx(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/problems", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: signed})
	h.Middleware(next).ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, gotClaims)
	assert.Equal(t, "user123", gotClaims.Sub)
	assert.Equal(t, "test@example.com", gotClaims.Email)
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	h := &Handler{jwtSecret: secret}

	claims := &Claims{
		Sub: "user123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // expired
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	require.NoError(t, err)

	var gotClaims *Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = ClaimsFromCtx(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/problems", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: signed})
	h.Middleware(next).ServeHTTP(httptest.NewRecorder(), req)

	assert.Nil(t, gotClaims)
}

func TestRequireAuth_Unauthorized(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/clusters", nil)
	w := httptest.NewRecorder()

	RequireAuth(next).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleLogout(t *testing.T) {
	h := &Handler{jwtSecret: []byte("secret")}
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	h.HandleLogout(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == cookieName {
			found = true
			assert.Equal(t, -1, c.MaxAge)
		}
	}
	assert.True(t, found, "session cookie should be cleared")
}

func TestHandleMe_NoSession(t *testing.T) {
	h := &Handler{jwtSecret: []byte("secret")}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	w := httptest.NewRecorder()
	h.HandleMe(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

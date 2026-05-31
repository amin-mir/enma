package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/amin-mir/enma/internal/auth"
	"github.com/amin-mir/enma/internal/handler/mocks"
)

func newTestApp(h *AuthHandler) *fiber.App {
	app := fiber.New()
	h.Mount(app)
	return app
}

func newAuthDeps(t *testing.T) *mocks.MockAuthService {
	ctrl := gomock.NewController(t)
	return mocks.NewMockAuthService(ctrl)
}

func doRequest(app *fiber.App, req *http.Request) *http.Response {
	resp, err := app.Test(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func TestAuthHandler_Register(t *testing.T) {
	t.Parallel()

	tokens := auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}

	tests := []struct {
		name   string
		body   func() *bytes.Reader
		setup  func(*mocks.MockAuthService)
		status int
		resp   any
	}{
		{
			name:   "invalid json",
			body:   func() *bytes.Reader { return bytes.NewReader([]byte("invalid")) },
			setup:  func(m *mocks.MockAuthService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"invalid request"},
		},
		{
			name:   "missing email",
			body:   func() *bytes.Reader { return jsonBody(t, registerReq{Password: "pass"}) },
			setup:  func(m *mocks.MockAuthService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"email and password are required"},
		},
		{
			name:   "missing password",
			body:   func() *bytes.Reader { return jsonBody(t, registerReq{Email: "a@b.com"}) },
			setup:  func(m *mocks.MockAuthService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"email and password are required"},
		},
		{
			name: "email already in use",
			body: func() *bytes.Reader {
				return jsonBody(t, registerReq{Email: "a@b.com", Password: "pass"})
			},
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Register(gomock.Any(), "a@b.com", "pass").Return(auth.TokenPair{}, auth.ErrEmailInUse)
			},
			status: fiber.StatusConflict,
			resp:   errResp{"email already in use"},
		},
		{
			name: "success",
			body: func() *bytes.Reader {
				return jsonBody(t, registerReq{Email: "a@b.com", Password: "pass"})
			},
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Register(gomock.Any(), "a@b.com", "pass").Return(tokens, nil)
			},
			status: fiber.StatusCreated,
			resp:   tokenResp{tokens.AccessToken, tokens.RefreshToken},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newAuthDeps(t)
			tt.setup(svc)
			app := newTestApp(NewAuthHandler(svc))

			req := httptest.NewRequest(http.MethodPost, "/auth/register", tt.body())
			req.Header.Set("Content-Type", "application/json")
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			requireJSON(t, resp, tt.resp)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	tokens := auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}

	tests := []struct {
		name   string
		body   func() *bytes.Reader
		setup  func(*mocks.MockAuthService)
		status int
		resp   any
	}{
		{
			name:   "invalid json",
			body:   func() *bytes.Reader { return bytes.NewReader([]byte("invalid")) },
			setup:  func(m *mocks.MockAuthService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"invalid request"},
		},
		{
			name: "invalid credentials",
			body: func() *bytes.Reader {
				return jsonBody(t, loginReq{Email: "a@b.com", Password: "wrong"})
			},
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Login(gomock.Any(), "a@b.com", "wrong").Return(auth.TokenPair{}, auth.ErrInvalidCredentials)
			},
			status: fiber.StatusUnauthorized,
			resp:   errResp{"invalid credentials"},
		},
		{
			name: "success",
			body: func() *bytes.Reader {
				return jsonBody(t, loginReq{Email: "a@b.com", Password: "pass"})
			},
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Login(gomock.Any(), "a@b.com", "pass").Return(tokens, nil)
			},
			status: fiber.StatusOK,
			resp:   tokenResp{tokens.AccessToken, tokens.RefreshToken},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newAuthDeps(t)
			tt.setup(svc)
			app := newTestApp(NewAuthHandler(svc))

			req := httptest.NewRequest(http.MethodPost, "/auth/login", tt.body())
			req.Header.Set("Content-Type", "application/json")
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			requireJSON(t, resp, tt.resp)
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Parallel()

	tokens := auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}

	tests := []struct {
		name   string
		body   func() *bytes.Reader
		setup  func(*mocks.MockAuthService)
		status int
		resp   any
	}{
		{
			name:   "missing refresh token",
			body:   func() *bytes.Reader { return jsonBody(t, refreshReq{}) },
			setup:  func(m *mocks.MockAuthService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"refresh_token is required"},
		},
		{
			name: "invalid token",
			body: func() *bytes.Reader { return jsonBody(t, refreshReq{RefreshToken: "bad"}) },
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Refresh(gomock.Any(), "bad").Return(auth.TokenPair{}, auth.ErrInvalidToken)
			},
			status: fiber.StatusUnauthorized,
			resp:   errResp{"invalid or expired refresh token"},
		},
		{
			name: "success",
			body: func() *bytes.Reader { return jsonBody(t, refreshReq{RefreshToken: "valid"}) },
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Refresh(gomock.Any(), "valid").Return(tokens, nil)
			},
			status: fiber.StatusOK,
			resp:   tokenResp{tokens.AccessToken, tokens.RefreshToken},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newAuthDeps(t)
			tt.setup(svc)
			app := newTestApp(NewAuthHandler(svc))

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", tt.body())
			req.Header.Set("Content-Type", "application/json")
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			requireJSON(t, resp, tt.resp)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		body   func() *bytes.Reader
		setup  func(*mocks.MockAuthService)
		status int
	}{
		{
			name:   "missing refresh token",
			body:   func() *bytes.Reader { return jsonBody(t, logoutReq{}) },
			setup:  func(m *mocks.MockAuthService) {},
			status: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: func() *bytes.Reader { return jsonBody(t, logoutReq{RefreshToken: "valid"}) },
			setup: func(m *mocks.MockAuthService) {
				m.EXPECT().Logout(gomock.Any(), "valid")
			},
			status: fiber.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newAuthDeps(t)
			tt.setup(svc)
			app := newTestApp(NewAuthHandler(svc))

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", tt.body())
			req.Header.Set("Content-Type", "application/json")
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func requireJSON(t *testing.T, resp *http.Response, want any) {
	t.Helper()
	wantBytes, err := json.Marshal(want)
	require.NoError(t, err)
	var wantMap, gotMap map[string]any
	require.NoError(t, json.Unmarshal(wantBytes, &wantMap))
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&gotMap))
	require.Equal(t, wantMap, gotMap)
}

package handler

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/amin-mir/enma/internal/auth"
)

//go:generate go tool mockgen -source=auth.go -destination=mocks/mock_auth.go -package=mocks AuthService
type AuthService interface {
	Register(ctx context.Context, email, pass string) (auth.TokenPair, error)
	Login(ctx context.Context, email, pass string) (auth.TokenPair, error)
	Refresh(ctx context.Context, plainToken string) (auth.TokenPair, error)
	Logout(ctx context.Context, plainToken string)
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type errResp struct {
	Error string `json:"error"`
}

type AuthHandler struct {
	auth AuthService
}

func NewAuthHandler(a AuthService) *AuthHandler {
	return &AuthHandler{auth: a}
}

func (h *AuthHandler) Mount(r fiber.Router) {
	g := r.Group("/auth")
	g.Post("/register", h.register)
	g.Post("/login", h.login)
	g.Post("/refresh", h.refresh)
	g.Post("/logout", h.logout)
}

func (h *AuthHandler) register(c fiber.Ctx) error {
	var req registerReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid request"})
	}
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"email and password are required"})
	}

	tokens, err := h.auth.Register(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrEmailInUse) {
			return c.Status(fiber.StatusConflict).JSON(errResp{"email already in use"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(tokenResp{tokens.AccessToken, tokens.RefreshToken})
}

func (h *AuthHandler) login(c fiber.Ctx) error {
	var req loginReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid request"})
	}

	tokens, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"invalid credentials"})
	}

	return c.JSON(tokenResp{tokens.AccessToken, tokens.RefreshToken})
}

func (h *AuthHandler) refresh(c fiber.Ctx) error {
	var req refreshReq
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"refresh_token is required"})
	}

	tokens, err := h.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"invalid or expired refresh token"})
	}

	return c.JSON(tokenResp{tokens.AccessToken, tokens.RefreshToken})
}

func (h *AuthHandler) logout(c fiber.Ctx) error {
	var req logoutReq
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"refresh_token is required"})
	}

	h.auth.Logout(c.Context(), req.RefreshToken)
	return c.SendStatus(fiber.StatusNoContent)
}

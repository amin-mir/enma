package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/amin-mir/enma/internal/auth"
)

type AuthHandler struct {
	auth *auth.Auth
}

func NewAuthHandler(a *auth.Auth) *AuthHandler {
	return &AuthHandler{auth: a}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email and password are required"})
	}

	tokens, err := h.auth.Register(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrEmailInUse) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already in use"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	tokens, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	return c.JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "refresh_token is required"})
	}

	tokens, err := h.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired refresh token"})
	}

	return c.JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "refresh_token is required"})
	}

	h.auth.Logout(c.Context(), req.RefreshToken)
	return c.SendStatus(fiber.StatusNoContent)
}

package auth

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/amin-mir/enma/internal/user"
)

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type errResp struct {
	Error string `json:"error"`
}

func (a *AuthService) JWTMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed token"})
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := a.validateAccessToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals("user_id", claims.UserID)
		return c.Next()
	}
}

func UserIDFromCtx(c fiber.Ctx) (uuid.UUID, error) {
	idStr, ok := c.Locals("user_id").(string)
	if !ok {
		return uuid.UUID{}, errors.New("user_id not in context")
	}
	return uuid.Parse(idStr)
}

func (a *AuthService) MountHTTPHandlers(r fiber.Router) {
	g := r.Group("/auth")
	g.Post("/register", a.register)
	g.Post("/login", a.login)
	g.Post("/refresh", a.refresh)
	g.Post("/logout", a.logout)
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *AuthService) register(c fiber.Ctx) error {
	var req registerReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid request"})
	}
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"email and password are required"})
	}

	tokens, err := a.Register(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrEmailInUse) {
			return c.Status(fiber.StatusConflict).JSON(errResp{"email already in use"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(tokenResp{tokens.AccessToken, tokens.RefreshToken})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *AuthService) login(c fiber.Ctx) error {
	var req loginReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid request"})
	}

	tokens, err := a.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"invalid credentials"})
	}

	return c.JSON(tokenResp{tokens.AccessToken, tokens.RefreshToken})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (a *AuthService) refresh(c fiber.Ctx) error {
	var req refreshReq
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"refresh_token is required"})
	}

	tokens, err := a.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"invalid or expired refresh token"})
	}

	return c.JSON(tokenResp{tokens.AccessToken, tokens.RefreshToken})
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (a *AuthService) logout(c fiber.Ctx) error {
	var req logoutReq
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"refresh_token is required"})
	}

	a.Logout(c.Context(), req.RefreshToken)
	return c.SendStatus(fiber.StatusNoContent)
}

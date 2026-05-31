package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/amin-mir/enma/internal/postgres"
)

type JournalHandler struct {
	pg *postgres.Postgres
}

func NewJournalHandler(pg *postgres.Postgres) *JournalHandler {
	return &JournalHandler{pg: pg}
}

func userIDFromCtx(c fiber.Ctx) (uuid.UUID, error) {
	idStr, ok := c.Locals("user_id").(string)
	if !ok {
		return uuid.UUID{}, errors.New("user_id not in context")
	}
	return uuid.Parse(idStr)
}

func (h *JournalHandler) Create(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
	}

	res, err := h.pg.CreateJournalEntry(c.Context(), userID, req.Content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         res.ID,
		"created_at": res.CreatedAt,
	})
}

func (h *JournalHandler) List(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	entries, err := h.pg.ListJournalEntries(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(entries)
}

func (h *JournalHandler) Get(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	entry, err := h.pg.GetJournalEntry(c.Context(), entryID, userID)
	if errors.Is(err, postgres.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(entry)
}

func (h *JournalHandler) Update(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
	}

	err = h.pg.UpdateJournalEntry(c.Context(), entryID, userID, req.Content)
	if errors.Is(err, postgres.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

package handler

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/amin-mir/enma/internal/model"
	"github.com/amin-mir/enma/internal/postgres"
)

//go:generate go tool mockgen -source=journal.go -destination=mocks/mock_journal.go -package=mocks JournalService
type JournalService interface {
	CreateJournalEntry(ctx context.Context, userID uuid.UUID, content string) (postgres.CreateJournalEntryResult, error)
	ListJournalEntries(ctx context.Context, userID uuid.UUID) ([]model.JournalEntry, error)
	GetJournalEntry(ctx context.Context, entryID, userID uuid.UUID) (model.JournalEntry, error)
	UpdateJournalEntry(ctx context.Context, entryID, userID uuid.UUID, content string, version int32) (int32, error)
}

type createJournalReq struct {
	Content string `json:"content"`
}

type createJournalResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Version   int32     `json:"version"`
}

type updateJournalReq struct {
	Content string `json:"content"`
	Version int32  `json:"version"`
}

type updateJournalResp struct {
	Version int32 `json:"version"`
}

type JournalHandler struct {
	svc JournalService
}

func NewJournalHandler(svc JournalService) *JournalHandler {
	return &JournalHandler{svc: svc}
}

func (h *JournalHandler) Mount(r fiber.Router, protected fiber.Handler) {
	g := r.Group("/journals", protected)
	g.Post("/", h.create)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Put("/:id", h.update)
}

func userIDFromCtx(c fiber.Ctx) (uuid.UUID, error) {
	idStr, ok := c.Locals("user_id").(string)
	if !ok {
		return uuid.UUID{}, errors.New("user_id not in context")
	}
	return uuid.Parse(idStr)
}

func (h *JournalHandler) create(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	var req createJournalReq
	if err := c.Bind().Body(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"content is required"})
	}

	res, err := h.svc.CreateJournalEntry(c.Context(), userID, req.Content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(createJournalResp{ID: res.ID, CreatedAt: res.CreatedAt, Version: res.Version})
}

func (h *JournalHandler) list(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	entries, err := h.svc.ListJournalEntries(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.JSON(entries)
}

func (h *JournalHandler) get(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid id"})
	}

	entry, err := h.svc.GetJournalEntry(c.Context(), entryID, userID)
	if errors.Is(err, postgres.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(errResp{"not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.JSON(entry)
}

func (h *JournalHandler) update(c fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid id"})
	}

	var req updateJournalReq
	if err := c.Bind().Body(&req); err != nil || req.Content == "" || req.Version <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"content and version are required"})
	}

	newVersion, err := h.svc.UpdateJournalEntry(c.Context(), entryID, userID, req.Content, req.Version)
	if errors.Is(err, postgres.ErrConflict) {
		return c.Status(fiber.StatusConflict).JSON(errResp{"conflict"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.JSON(updateJournalResp{Version: newVersion})
}

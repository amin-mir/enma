package journal

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/amin-mir/enma/internal/auth"
)

type errResp struct {
	Error string `json:"error"`
}

func (s *JournalService) MountHTTPHandlers(r fiber.Router, authMiddleware fiber.Handler) {
	g := r.Group("/journals", authMiddleware)
	g.Post("/", s.create)
	g.Get("/", s.list)
	g.Get("/:id", s.get)
	g.Put("/:id", s.update)
}

type createReq struct {
	Content string `json:"content"`
}

type createResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Version   int32     `json:"version"`
}

func (s *JournalService) create(c fiber.Ctx) error {
	userID, err := auth.UserIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	var req createReq
	if err := c.Bind().Body(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"content is required"})
	}

	st := Store{DB: s.pool}
	res, err := st.Create(c.Context(), userID, req.Content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(createResp{ID: res.ID, CreatedAt: res.CreatedAt, Version: res.Version})
}

func (s *JournalService) list(c fiber.Ctx) error {
	userID, err := auth.UserIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	st := Store{DB: s.pool}
	entries, err := st.List(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.JSON(entries)
}

func (s *JournalService) get(c fiber.Ctx) error {
	userID, err := auth.UserIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid id"})
	}

	st := Store{DB: s.pool}
	entry, err := st.Get(c.Context(), entryID, userID)
	if errors.Is(err, ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(errResp{"not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.JSON(entry)
}

type updateReq struct {
	Content string `json:"content"`
	Version int32  `json:"version"`
}

type updateResp struct {
	Version int32 `json:"version"`
}

func (s *JournalService) update(c fiber.Ctx) error {
	userID, err := auth.UserIDFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResp{"unauthorized"})
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"invalid id"})
	}

	var req updateReq
	if err := c.Bind().Body(&req); err != nil || req.Content == "" || req.Version <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(errResp{"content and version are required"})
	}

	st := Store{DB: s.pool}
	newVersion, err := st.Update(c.Context(), entryID, userID, req.Content, req.Version)
	if errors.Is(err, ErrConflict) {
		return c.Status(fiber.StatusConflict).JSON(errResp{"conflict"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResp{"internal error"})
	}

	return c.JSON(updateResp{Version: newVersion})
}

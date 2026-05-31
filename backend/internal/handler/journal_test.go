package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/amin-mir/enma/internal/handler/mocks"
	"github.com/amin-mir/enma/internal/model"
	"github.com/amin-mir/enma/internal/postgres"
)

func newJournalTestApp(h *JournalHandler, userID uuid.UUID) *fiber.App {
	app := fiber.New()
	injectUser := func(c fiber.Ctx) error {
		c.Locals("user_id", userID.String())
		return c.Next()
	}
	h.Mount(app, injectUser)
	return app
}

func newJournalDeps(t *testing.T) *mocks.MockJournalService {
	ctrl := gomock.NewController(t)
	return mocks.NewMockJournalService(ctrl)
}

func TestJournalHandler_Create(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)
	entryID := uuid.New()

	tests := []struct {
		name   string
		body   func(t *testing.T) []byte
		setup  func(*mocks.MockJournalService)
		status int
		resp   any
	}{
		{
			name:   "invalid json",
			body:   func(t *testing.T) []byte { return []byte("invalid") },
			setup:  func(m *mocks.MockJournalService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"content is required"},
		},
		{
			name:   "missing content",
			body:   func(t *testing.T) []byte { return []byte(`{}`) },
			setup:  func(m *mocks.MockJournalService) {},
			status: fiber.StatusBadRequest,
			resp:   errResp{"content is required"},
		},
		{
			name: "DB error",
			body: func(t *testing.T) []byte {
				b, err := json.Marshal(createJournalReq{Content: "hello"})
				require.NoError(t, err)
				return b
			},
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().CreateJournalEntry(gomock.Any(), userID, "hello").Return(postgres.CreateJournalEntryResult{}, errors.New("db error"))
			},
			status: fiber.StatusInternalServerError,
			resp:   errResp{"internal error"},
		},
		{
			name: "success",
			body: func(t *testing.T) []byte {
				b, err := json.Marshal(createJournalReq{Content: "hello"})
				require.NoError(t, err)
				return b
			},
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().CreateJournalEntry(gomock.Any(), userID, "hello").Return(postgres.CreateJournalEntryResult{ID: entryID, CreatedAt: now}, nil)
			},
			status: fiber.StatusCreated,
			resp:   createJournalResp{ID: entryID, CreatedAt: now},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newJournalDeps(t)
			tt.setup(svc)
			app := newJournalTestApp(NewJournalHandler(svc), userID)

			req := httptest.NewRequest(http.MethodPost, "/journals/", bytes.NewReader(tt.body(t)))
			req.Header.Set("Content-Type", "application/json")
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			requireJSON(t, resp, tt.resp)
		})
	}
}

func TestJournalHandler_List(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entries := []model.JournalEntry{
		{ID: uuid.New(), UserID: userID, Content: "entry 1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), UserID: userID, Content: "entry 2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	tests := []struct {
		name   string
		setup  func(*mocks.MockJournalService)
		status int
		count  int
	}{
		{
			name: "DB error",
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().ListJournalEntries(gomock.Any(), userID).Return(nil, errors.New("db error"))
			},
			status: fiber.StatusInternalServerError,
		},
		{
			name: "empty list",
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().ListJournalEntries(gomock.Any(), userID).Return([]model.JournalEntry{}, nil)
			},
			status: fiber.StatusOK,
			count:  0,
		},
		{
			name: "success",
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().ListJournalEntries(gomock.Any(), userID).Return(entries, nil)
			},
			status: fiber.StatusOK,
			count:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newJournalDeps(t)
			tt.setup(svc)
			app := newJournalTestApp(NewJournalHandler(svc), userID)

			req := httptest.NewRequest(http.MethodGet, "/journals/", nil)
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			if tt.status == fiber.StatusOK {
				var got []model.JournalEntry
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
				require.Len(t, got, tt.count)
			}
		})
	}
}

func TestJournalHandler_Get(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	entry := model.JournalEntry{ID: entryID, UserID: userID, Content: "hello", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	tests := []struct {
		name   string
		id     string
		setup  func(*mocks.MockJournalService)
		status int
	}{
		{
			name:   "invalid id",
			id:     "not-a-uuid",
			setup:  func(m *mocks.MockJournalService) {},
			status: fiber.StatusBadRequest,
		},
		{
			name: "not found",
			id:   entryID.String(),
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().GetJournalEntry(gomock.Any(), entryID, userID).Return(model.JournalEntry{}, postgres.ErrNotFound)
			},
			status: fiber.StatusNotFound,
		},
		{
			name: "DB error",
			id:   entryID.String(),
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().GetJournalEntry(gomock.Any(), entryID, userID).Return(model.JournalEntry{}, errors.New("db error"))
			},
			status: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			id:   entryID.String(),
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().GetJournalEntry(gomock.Any(), entryID, userID).Return(entry, nil)
			},
			status: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newJournalDeps(t)
			tt.setup(svc)
			app := newJournalTestApp(NewJournalHandler(svc), userID)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/journals/%s", tt.id), nil)
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestJournalHandler_Update(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name   string
		id     string
		body   func(t *testing.T) []byte
		setup  func(*mocks.MockJournalService)
		status int
	}{
		{
			name:  "invalid id",
			id:    "not-a-uuid",
			body:  func(t *testing.T) []byte { return []byte(`{"content":"hello"}`) },
			setup: func(m *mocks.MockJournalService) {},
			status: fiber.StatusBadRequest,
		},
		{
			name:  "missing content",
			id:    entryID.String(),
			body:  func(t *testing.T) []byte { return []byte(`{}`) },
			setup: func(m *mocks.MockJournalService) {},
			status: fiber.StatusBadRequest,
		},
		{
			name: "not found",
			id:   entryID.String(),
			body: func(t *testing.T) []byte {
				b, err := json.Marshal(updateJournalReq{Content: "hello"})
				require.NoError(t, err)
				return b
			},
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().UpdateJournalEntry(gomock.Any(), entryID, userID, "hello").Return(postgres.ErrNotFound)
			},
			status: fiber.StatusNotFound,
		},
		{
			name: "DB error",
			id:   entryID.String(),
			body: func(t *testing.T) []byte {
				b, err := json.Marshal(updateJournalReq{Content: "hello"})
				require.NoError(t, err)
				return b
			},
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().UpdateJournalEntry(gomock.Any(), entryID, userID, "hello").Return(errors.New("db error"))
			},
			status: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			id:   entryID.String(),
			body: func(t *testing.T) []byte {
				b, err := json.Marshal(updateJournalReq{Content: "hello"})
				require.NoError(t, err)
				return b
			},
			setup: func(m *mocks.MockJournalService) {
				m.EXPECT().UpdateJournalEntry(gomock.Any(), entryID, userID, "hello").Return(nil)
			},
			status: fiber.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newJournalDeps(t)
			tt.setup(svc)
			app := newJournalTestApp(NewJournalHandler(svc), userID)

			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/journals/%s", tt.id), bytes.NewReader(tt.body(t)))
			req.Header.Set("Content-Type", "application/json")
			resp := doRequest(app, req)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

package testutil_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/amin-mir/enma/internal/testutil"
)

type item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newApp() *fiber.App {
	app := fiber.New()

	app.Post("/echo", func(c fiber.Ctx) error {
		var body item
		_ = c.Bind().Body(&body)
		return c.JSON(body)
	})

	app.Get("/items", func(c fiber.Ctx) error {
		return c.JSON([]item{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}})
	})

	app.Post("/no-content", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Post("/content-type-check", func(c fiber.Ctx) error {
		if c.Get("Content-Type") != "application/json" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{})
	})

	app.Get("/no-content-type-check", func(c fiber.Ctx) error {
		if c.Get("Content-Type") != "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{})
	})

	return app
}

func TestRequest_DecodesResponseIntoConcreteType(t *testing.T) {
	app := newApp()

	status, got, err := testutil.Request[item](app, http.MethodPost, "/echo", item{ID: 42, Name: "hello"})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, status)
	require.Equal(t, item{ID: 42, Name: "hello"}, got)
}

func TestRequest_DecodesSliceResponse(t *testing.T) {
	// Use []T as the type parameter for list endpoints.
	app := newApp()

	status, got, err := testutil.Request[[]item](app, http.MethodGet, "/items", nil)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, status)
	require.Len(t, got, 2)
	require.Equal(t, item{ID: 1, Name: "one"}, got[0])
}

func TestRequest_StructEmptyIgnoresResponseBody(t *testing.T) {
	// Use struct{} when only the status code matters.
	// The response body is read and discarded without error.
	app := newApp()

	status, _, err := testutil.Request[struct{}](app, http.MethodPost, "/echo", item{ID: 1})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, status)
}

func TestRequest_EmptyResponseBody(t *testing.T) {
	// 204 No Content has no body. The decoder returns io.EOF, which Request
	// treats as a non-error — empty body is a valid response.
	// Use struct{} as T since there is nothing to decode.
	app := newApp()

	status, _, err := testutil.Request[struct{}](app, http.MethodPost, "/no-content", nil)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNoContent, status)
}

func TestRequest_NilBodySkipsContentType(t *testing.T) {
	// Passing nil body (e.g. for GET) does not set Content-Type.
	app := newApp()

	status, _, err := testutil.Request[struct{}](app, http.MethodGet, "/no-content-type-check", nil)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, status)
}

func TestRequest_SetsContentTypeForJSONBody(t *testing.T) {
	// When body is non-nil, Content-Type: application/json is set automatically.
	app := newApp()

	status, _, err := testutil.Request[struct{}](app, http.MethodPost, "/content-type-check", item{ID: 1})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, status)
}

func TestRequest_MarshalError(t *testing.T) {
	// Passing an unmarshalable type (e.g. channel) returns an error immediately
	// before the request is sent.
	app := newApp()

	_, _, err := testutil.Request[struct{}](app, http.MethodPost, "/echo", make(chan int))
	require.Error(t, err)
}

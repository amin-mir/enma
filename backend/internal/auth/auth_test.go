package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/amin-mir/enma/internal/model"
	"github.com/amin-mir/enma/internal/password"
	"github.com/amin-mir/enma/internal/postgres"
	"github.com/amin-mir/enma/internal/postgres/mocks"
)

func newTestAuth(t *testing.T, db postgres.DB) *Auth {
	t.Helper()
	return New(db, zerolog.Nop(), Config{
		Secret:               "test-secret",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 30 * 24 * time.Hour,
	})
}

func TestRegister(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	userID := uuid.New()
	db.EXPECT().CreateUser(ctx, "user@example.com", gomock.Any()).Return(userID, nil)
	db.EXPECT().CreateRefreshToken(ctx, userID, gomock.Any(), gomock.Any()).Return(nil)

	tokens, err := newTestAuth(t, db).Register(ctx, "user@example.com", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	db.EXPECT().CreateUser(ctx, "dupe@example.com", gomock.Any()).Return(uuid.UUID{}, postgres.ErrUniqueViolation)

	_, err := newTestAuth(t, db).Register(ctx, "dupe@example.com", "password123")
	require.ErrorIs(t, err, ErrEmailInUse)
}

func TestLogin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	// Use a real argon2 hash of "password123".
	hash, err := password.Hash("password123")
	require.NoError(t, err)

	userID := uuid.New()
	db.EXPECT().GetUserByEmail(ctx, "user@example.com").Return(model.User{
		ID:           userID,
		Email:        "user@example.com",
		PasswordHash: hash,
	}, nil)
	db.EXPECT().CreateRefreshToken(ctx, userID, gomock.Any(), gomock.Any()).Return(nil)

	tokens, err := newTestAuth(t, db).Login(ctx, "user@example.com", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	db.EXPECT().GetUserByEmail(ctx, "nobody@example.com").Return(model.User{}, postgres.ErrNotFound)

	_, err := newTestAuth(t, db).Login(ctx, "nobody@example.com", "password123")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	hash, err := password.Hash("correct-password")
	require.NoError(t, err)

	db.EXPECT().GetUserByEmail(ctx, "user@example.com").Return(model.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: hash,
	}, nil)

	_, err = newTestAuth(t, db).Login(ctx, "user@example.com", "wrong-password")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_MalformedHash(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	db.EXPECT().GetUserByEmail(ctx, "user@example.com").Return(model.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: "not-a-valid-hash",
	}, nil)

	_, err := newTestAuth(t, db).Login(ctx, "user@example.com", "password123")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	db.EXPECT().RotateRefreshToken(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(uuid.New(), nil)

	tokens, err := newTestAuth(t, db).Refresh(ctx, "some-plain-token")
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
}

func TestRefresh_InvalidToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	db.EXPECT().RotateRefreshToken(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(uuid.UUID{}, postgres.ErrNotFound)

	_, err := newTestAuth(t, db).Refresh(ctx, "invalid-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestLogout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	db.EXPECT().DeleteRefreshTokenByHash(ctx, gomock.Any()).Return(nil)

	newTestAuth(t, db).Logout(ctx, "some-plain-token")
}

func TestValidateAccessToken(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)
	a := newTestAuth(t, db)

	userID := uuid.New()
	tokenStr, err := a.newAccessToken(userID)
	require.NoError(t, err)

	claims, err := a.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	require.Equal(t, userID.String(), claims.UserID)
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	_, err := newTestAuth(t, db).ValidateAccessToken("not.a.valid.token")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := mocks.NewMockDB(ctrl)

	// Token signed with a different secret.
	other := New(db, zerolog.Nop(), Config{
		Secret:              "other-secret",
		AccessTokenDuration: 15 * time.Minute,
	})
	tokenStr, err := other.newAccessToken(uuid.New())
	require.NoError(t, err)

	_, err = newTestAuth(t, db).ValidateAccessToken(tokenStr)
	require.ErrorIs(t, err, ErrInvalidToken)
}


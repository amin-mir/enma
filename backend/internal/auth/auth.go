package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/amin-mir/enma/internal/password"
	"github.com/amin-mir/enma/internal/postgres"
)

// dummyHash is used during login when the email doesn't exist, so that
// password verification always runs and response time stays constant.
const dummyHash = "$argon2id$v=19$m=65536,t=1,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrEmailInUse         = errors.New("email already in use")
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Config struct {
	Secret               string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

type Auth struct {
	db     postgres.DB
	logger zerolog.Logger
	cfg    Config
}

func New(db postgres.DB, logger zerolog.Logger, cfg Config) *Auth {
	return &Auth{db: db, logger: logger, cfg: cfg}
}

func (a *Auth) Register(ctx context.Context, email, pass string) (TokenPair, error) {
	hash, err := password.Hash(pass)
	if err != nil {
		return TokenPair{}, err
	}

	userID, err := a.db.CreateUser(ctx, email, hash)
	if err != nil {
		if errors.Is(err, postgres.ErrUniqueViolation) {
			return TokenPair{}, ErrEmailInUse
		}
		return TokenPair{}, err
	}

	return a.issueTokens(ctx, userID)
}

func (a *Auth) Login(ctx context.Context, email, pass string) (TokenPair, error) {
	user, dbErr := a.db.GetUserByEmail(ctx, email)

	// Always verify a hash to prevent timing-based email enumeration.
	hashToCheck := dummyHash
	if dbErr == nil {
		hashToCheck = user.PasswordHash
	}

	match, err := password.Verify(pass, hashToCheck)
	if err != nil {
		a.logger.Error().Err(err).Msg("password verification failed")
	}
	if dbErr != nil || !match {
		return TokenPair{}, ErrInvalidCredentials
	}

	return a.issueTokens(ctx, user.ID)
}

func (a *Auth) Refresh(ctx context.Context, plainToken string) (TokenPair, error) {
	newPlain, newHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	userID, err := a.db.RotateRefreshToken(ctx, hashToken(plainToken), newHash, time.Now().Add(a.cfg.RefreshTokenDuration))
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}

	accessToken, err := a.newAccessToken(userID)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: newPlain}, nil
}

func (a *Auth) Logout(ctx context.Context, plainToken string) {
	if err := a.db.DeleteRefreshTokenByHash(ctx, hashToken(plainToken)); err != nil {
		a.logger.Error().Err(err).Msg("failed to delete refresh token on logout")
	}
}

func (a *Auth) ValidateAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(a.cfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (a *Auth) issueTokens(ctx context.Context, userID uuid.UUID) (TokenPair, error) {
	accessToken, err := a.newAccessToken(userID)
	if err != nil {
		return TokenPair{}, err
	}

	plain, hashed, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	err = a.db.CreateRefreshToken(ctx, userID, hashed, time.Now().Add(a.cfg.RefreshTokenDuration))
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: plain}, nil
}

func (a *Auth) newAccessToken(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.cfg.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(a.cfg.Secret))
}

func newRefreshToken() (plain, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = base64.URLEncoding.EncodeToString(b)
	hashed = hashToken(plain)
	return
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/amin-mir/enma/internal/auth/password"
	"github.com/amin-mir/enma/internal/user"
)

// dummyHash is used during login when the email doesn't exist, so that
// password verification always runs and response time stays constant.
const dummyHash = "$argon2id$v=19$m=65536,t=1,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
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

type AuthService struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
	cfg    Config
}

func New(pool *pgxpool.Pool, logger zerolog.Logger, cfg Config) *AuthService {
	return &AuthService{pool: pool, logger: logger, cfg: cfg}
}

func (a *AuthService) Register(ctx context.Context, email, pass string) (TokenPair, error) {
	hash, err := password.Hash(pass)
	if err != nil {
		return TokenPair{}, err
	}

	plain, hashed, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	expiresAt := time.Now().Add(a.cfg.RefreshTokenDuration)
	s := Store{DB: a.pool}
	userID, err := s.CreateUserAndRefreshToken(ctx, email, hash, hashed, expiresAt)
	if err != nil {
		return TokenPair{}, err
	}

	accessToken, err := a.newAccessToken(userID)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: plain}, nil
}

func (a *AuthService) Login(ctx context.Context, email, pass string) (TokenPair, error) {
	us := user.Store{DB: a.pool}
	u, dbErr := us.GetByEmail(ctx, email)

	// Always verify a hash to prevent timing-based email enumeration.
	hashToCheck := dummyHash
	if dbErr == nil {
		hashToCheck = u.PasswordHash
	}

	match, err := password.Verify(pass, hashToCheck)
	if err != nil {
		a.logger.Error().Err(err).Msg("password verification failed")
	}
	if dbErr != nil || !match {
		return TokenPair{}, ErrInvalidCredentials
	}

	accessToken, err := a.newAccessToken(u.ID)
	if err != nil {
		return TokenPair{}, err
	}

	plain, hashed, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	expiresAt := time.Now().Add(a.cfg.RefreshTokenDuration)
	s := Store{DB: a.pool}
	err = s.CreateRefreshToken(ctx, u.ID, hashed, expiresAt)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: plain}, nil
}

func (a *AuthService) Refresh(ctx context.Context, plainToken string) (TokenPair, error) {
	newPlain, newHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	expiresAt := time.Now().Add(a.cfg.RefreshTokenDuration)
	s := Store{DB: a.pool}
	userID, err := s.RotateRefreshToken(ctx, hashToken(plainToken), newHash, expiresAt)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}

	accessToken, err := a.newAccessToken(userID)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: newPlain}, nil
}

func (a *AuthService) Logout(ctx context.Context, plainToken string) {
	s := Store{DB: a.pool}
	err := s.DeleteRefreshTokenByHash(ctx, hashToken(plainToken))
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to delete refresh token on logout")
	}
}

func (a *AuthService) validateAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) {
			return []byte(a.cfg.Secret), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}


func (a *AuthService) newAccessToken(userID uuid.UUID) (string, error) {
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

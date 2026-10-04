package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const issuer = "particleapp"

var ErrInvalidToken = errors.New("invalid token")

type Config struct {
	Secret        string
	TTL           time.Duration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// Claims — проверенные данные пользователя внутри JWT.
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// Manager выпускает JWT, проверяет подпись и срок действия и хранит
// отозванные идентификаторы токенов в Redis до окончания их TTL.
type Manager struct {
	secret []byte
	ttl    time.Duration
	redis  *redis.Client
}

func New(ctx context.Context, cfg Config) (*Manager, error) {
	if cfg.Secret == "" {
		return nil, errors.New("JWT secret is empty")
	}
	if cfg.TTL <= 0 {
		return nil, errors.New("JWT TTL must be positive")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}

	return &Manager{secret: []byte(cfg.Secret), ttl: cfg.TTL, redis: client}, nil
}

func (m *Manager) Issue(userID uint) (string, error) {
	now := time.Now()
	jti, err := randomID()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) Parse(ctx context.Context, encoded string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(encoded, claims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID == 0 || claims.ID == "" {
		return nil, ErrInvalidToken
	}

	revoked, err := m.redis.Exists(ctx, blacklistKey(claims.ID)).Result()
	if err != nil {
		return nil, fmt.Errorf("проверка blacklist: %w", err)
	}
	if revoked > 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *Manager) Revoke(ctx context.Context, claims *Claims) error {
	if claims == nil || claims.ID == "" || claims.ExpiresAt == nil {
		return ErrInvalidToken
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	value := fmt.Sprintf("user_id=%d", claims.UserID)
	if err := m.redis.Set(ctx, blacklistKey(claims.ID), value, ttl).Err(); err != nil {
		return fmt.Errorf("добавление JWT в blacklist: %w", err)
	}
	return nil
}

func (m *Manager) Ping(ctx context.Context) error {
	return m.redis.Ping(ctx).Err()
}

func (m *Manager) Close() error {
	return m.redis.Close()
}

func blacklistKey(jti string) string {
	return "jwt:blacklist:" + jti
}

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

type contextKey struct{}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(contextKey{}).(*Claims)
	return claims, ok && claims != nil
}

func UserID(ctx context.Context) (uint, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.UserID == 0 {
		return 0, false
	}
	return claims.UserID, true
}

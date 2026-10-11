package user

import (
	"context"

	"particleapp/internal/auth"
	"particleapp/internal/models"
)

// Repository — порт хранения пользователей (реализация — internal/storage/postgres).
type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
}

// Domain — бизнес-логика домена «пользователь»:
// POST /api/users, POST /api/users/login, POST /api/users/logout.
type Domain interface {
	Register(ctx context.Context, request RegisterRequest) (*UserResponse, error)
	Login(ctx context.Context, request LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context) error
}

type Tokens interface {
	Issue(userID uint) (string, error)
	Revoke(ctx context.Context, claims *auth.Claims) error
}

type Usecase struct {
	repo   Repository
	tokens Tokens
}

func New(repo Repository, tokens Tokens) *Usecase {
	return &Usecase{repo: repo, tokens: tokens}
}

package user

import (
	"context"
	"errors"
	"strings"

	"particleapp/internal/apperror"
	"particleapp/internal/auth"
	"particleapp/internal/models"
	"particleapp/internal/storage"
)

const (
	minLoginLength    = 3
	maxLoginLength    = 50
	minPasswordLength = 6
	maxPasswordLength = 72
)

// Register — POST /api/users. Проверяет занятость логина и создаёт
// пользователя (пароль хранится как есть, по требованию лабораторной).
// Идентификатор и дата регистрации — на бэкенде.
func (u *Usecase) Register(ctx context.Context, request RegisterRequest) (*UserResponse, error) {
	login := strings.TrimSpace(request.Login)

	if err := validateLogin(login); err != nil {
		return nil, err
	}
	if err := validatePassword(request.Password); err != nil {
		return nil, err
	}

	if _, err := u.repo.GetUserByLogin(ctx, login); err == nil {
		return nil, apperror.Conflict("login_taken", "Пользователь с таким логином уже зарегистрирован")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, apperror.Internal("Не удалось проверить логин")
	}

	registered := models.User{Login: login, Password: request.Password}
	if err := u.repo.CreateUser(ctx, &registered); err != nil {
		if errors.Is(err, storage.ErrDuplicatedKey) {
			return nil, apperror.Conflict("login_taken", "Пользователь с таким логином уже зарегистрирован")
		}
		return nil, apperror.Internal("Не удалось зарегистрировать пользователя")
	}

	response := newUserResponse(registered)
	return &response, nil
}

// Login — POST /api/users/login. Проверяет логин и пароль и выдаёт подписанный JWT.
func (u *Usecase) Login(ctx context.Context, request LoginRequest) (*LoginResponse, error) {
	login := strings.TrimSpace(request.Login)
	if login == "" || request.Password == "" {
		return nil, apperror.Unauthorized("Неверный логин или пароль")
	}

	found, err := u.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, apperror.Unauthorized("Неверный логин или пароль")
		}
		return nil, apperror.Internal("Не удалось выполнить вход")
	}
	if found.Password != request.Password {
		return nil, apperror.Unauthorized("Неверный логин или пароль")
	}

	token, err := u.tokens.Issue(found.ID)
	if err != nil {
		return nil, apperror.Internal("Не удалось выпустить токен")
	}

	return &LoginResponse{
		Token: token,
		User:  newUserResponse(*found),
	}, nil
}

// Logout — POST /api/users/logout. Отзывает текущий JWT через Redis blacklist.
func (u *Usecase) Logout(ctx context.Context) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return apperror.Unauthorized("Требуется авторизация")
	}
	if err := u.tokens.Revoke(ctx, claims); err != nil {
		return apperror.Internal("Не удалось завершить сеанс")
	}
	return nil
}

func validateLogin(login string) error {
	length := len([]rune(login))
	if length < minLoginLength || length > maxLoginLength {
		return apperror.Validation("Логин должен быть от 3 до 50 символов")
	}
	if strings.ContainsAny(login, " \t\n") {
		return apperror.Validation("Логин не должен содержать пробелов")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return apperror.Validation("Пароль должен быть не короче 6 символов")
	}
	if len(password) > maxPasswordLength {
		return apperror.Validation("Пароль должен быть не длиннее 72 символов")
	}
	return nil
}

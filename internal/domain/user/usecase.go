package user

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"particleapp/internal/apperror"
	"particleapp/internal/auth"
	"particleapp/internal/models"
	"particleapp/internal/storage"
)

const (
	minUsernameLength = 3
	maxUsernameLength = 50
	minPasswordLength = 6
	maxPasswordLength = 72
	stubTokenPrefix   = "lab4-stub-token-"
)

// Register — POST /api/users. Проверяет занятость email, хеширует пароль
// и создаёт пользователя. Идентификатор и дата регистрации — на бэкенде.
func (u *Usecase) Register(ctx context.Context, request RegisterRequest) (*UserResponse, error) {
	username := strings.TrimSpace(request.Username)
	email := strings.ToLower(strings.TrimSpace(request.Email))

	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(request.Password); err != nil {
		return nil, err
	}

	if _, err := u.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, apperror.Conflict("email_taken", "Пользователь с таким email уже зарегистрирован")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, apperror.Internal("Не удалось проверить email")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal("Не удалось зашифровать пароль")
	}

	registered := models.User{Username: username, Email: email, PasswordHash: string(hash)}
	if err := u.repo.CreateUser(ctx, &registered); err != nil {
		if errors.Is(err, storage.ErrDuplicatedKey) {
			return nil, apperror.Conflict("email_taken", "Пользователь с таким email уже зарегистрирован")
		}
		return nil, apperror.Internal("Не удалось зарегистрировать пользователя")
	}

	response := newUserResponse(registered)
	return &response, nil
}

// Login — POST /api/users/login. Заглушка лабораторной работы 3:
// настоящая аутентификация, выдача и проверка токена — в лабораторной 4,
// поэтому учётные данные из тела запроса не проверяются, а пользователь
// берётся из функции-singleton.
func (u *Usecase) Login(ctx context.Context, request LoginRequest) (*LoginResponse, error) {
	currentUserID := auth.Get().ID

	found, err := u.repo.GetUserByID(ctx, currentUserID)
	if err != nil {
		return nil, apperror.Internal("Не удалось получить текущего пользователя")
	}

	return &LoginResponse{
		Token: stubTokenPrefix + strconv.FormatUint(uint64(currentUserID), 10),
		User:  newUserResponse(*found),
	}, nil
}

// Logout — POST /api/users/logout. Заглушка: сессии появятся в лабораторной 4.
func (u *Usecase) Logout(ctx context.Context) error {
	return nil
}

func validateUsername(username string) error {
	length := len([]rune(username))
	if length < minUsernameLength || length > maxUsernameLength {
		return apperror.Validation("Имя пользователя должно быть от 3 до 50 символов")
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" || len(email) > 100 || !strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@") {
		return apperror.Validation("Некорректный email")
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

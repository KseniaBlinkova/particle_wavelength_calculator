package user

import "particleapp/internal/models"

// UserResponse — сериализатор пользователя. Хеш пароля наружу не отдается.
type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// RegisterRequest — POST /api/users. Идентификатор и даты вычисляются на бэкенде.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest — POST /api/users/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse — JWT и данные аутентифицированного пользователя.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

func newUserResponse(user models.User) UserResponse {
	return UserResponse{ID: user.ID, Username: user.Username, Email: user.Email}
}

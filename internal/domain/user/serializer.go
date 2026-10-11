package user

import "particleapp/internal/models"

// UserResponse — сериализатор пользователя. Пароль наружу не отдается.
type UserResponse struct {
	ID    uint   `json:"id"`
	Login string `json:"login"`
}

// RegisterRequest — POST /api/users. Идентификатор и даты вычисляются на бэкенде.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginRequest — POST /api/users/login.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginResponse — JWT и данные аутентифицированного пользователя.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

func newUserResponse(user models.User) UserResponse {
	return UserResponse{ID: user.ID, Login: user.Login}
}

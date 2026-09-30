package web

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"particleapp/internal/apperror"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("web: ошибка сериализации ответа: %v", err)
	}
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		log.Printf("web: необработанная ошибка: %v", err)
		appErr = apperror.Internal("Внутренняя ошибка сервера")
	}
	w.WriteHeader(appErr.Status)
}

// DecodeJSON — разбор тела запроса. Неизвестные поля игнорируются, поэтому
// клиент не может подменить системные поля модели через JSON.
func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := decoder.Decode(dst); err != nil {
		return apperror.Validation("Некорректный JSON в теле запроса")
	}
	return nil
}

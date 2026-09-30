package user

import (
	"net/http"

	"particleapp/internal/web"
)

// Register — маршруты домена «пользователь». Все адреса начинаются с /api.
func Register(mux *http.ServeMux, domain Domain) {
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		register(domain)(w, r)
	})
	mux.HandleFunc("POST /api/users/login", func(w http.ResponseWriter, r *http.Request) {
		login(domain)(w, r)
	})
	mux.HandleFunc("POST /api/users/logout", func(w http.ResponseWriter, r *http.Request) {
		logout(domain)(w, r)
	})
}

// register — POST /api/users
func register(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request RegisterRequest
		if err := web.DecodeJSON(r, &request); err != nil {
			web.WriteError(w, err)
			return
		}

		response, err := domain.Register(r.Context(), request)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusCreated, response)
	}
}

// login — POST /api/users/login (заглушка)
func login(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request LoginRequest
		if err := web.DecodeJSON(r, &request); err != nil {
			web.WriteError(w, err)
			return
		}

		response, err := domain.Login(r.Context(), request)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, response)
	}
}

// logout — POST /api/users/logout (заглушка)
func logout(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := domain.Logout(r.Context()); err != nil {
			web.WriteError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"particleapp/internal/apperror"
	"particleapp/internal/auth"
	"particleapp/internal/domain/particle"
	"particleapp/internal/domain/user"
	"particleapp/internal/filestorage"
	"particleapp/internal/storage/postgres"
	"particleapp/internal/swagger"
	"particleapp/internal/web"
)

type Dependencies struct {
	Particles particle.Domain
	Users     user.Domain
	Store     *postgres.Store
	Files     *filestorage.MinIO
	Auth      *auth.Manager
}

// NewRouter отдаёт только REST API. HTML-шаблонов и server-rendered маршрутов
// в сервисе нет.
func NewRouter(deps Dependencies) http.Handler {
	api := http.NewServeMux()
	particle.Register(api, deps.Particles)
	user.Register(api, deps.Users)
	api.HandleFunc("GET /api/health", health(deps))
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	mux := http.NewServeMux()
	mux.Handle("/api/", middleware(api, deps.Auth))
	swagger.Register(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	return mux
}

func health(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if err := deps.Auth.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		payload := map[string]string{
			"api":    "/api",
			"bucket": deps.Files.Bucket(),
			"redis":  "ok",
		}
		web.WriteJSON(w, http.StatusOK, payload)
	}
}

func middleware(next http.Handler, tokens *auth.Manager) http.Handler {
	secured := authenticate(tokens, authorize(next))
	return recoverPanic(logRequests(withCORS(secured)))
}

func authenticate(tokens *auth.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			// Публичные методы работают независимо от токена: с некорректным
			// заголовком запрос обрабатывается как от гостя.
			if publicRoute(r) {
				next.ServeHTTP(w, r)
				return
			}
			web.WriteError(w, apperror.Unauthorized("Ожидается заголовок Authorization: Bearer <token>"))
			return
		}

		claims, err := tokens.Parse(r.Context(), parts[1])
		if err != nil {
			if publicRoute(r) && errors.Is(err, auth.ErrInvalidToken) {
				next.ServeHTTP(w, r)
				return
			}
			if errors.Is(err, auth.ErrInvalidToken) {
				web.WriteError(w, apperror.Unauthorized("JWT недействителен или отозван"))
			} else {
				web.WriteError(w, apperror.Internal("Не удалось проверить JWT"))
			}
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
	})
}

func authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicRoute(r) {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := auth.UserID(r.Context()); !ok {
			web.WriteError(w, apperror.Unauthorized("Требуется авторизация"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func publicRoute(r *http.Request) bool {
	if r.Method == http.MethodPost && (r.URL.Path == "/api/users" || r.URL.Path == "/api/users/login") {
		return true
	}
	if r.Method != http.MethodGet {
		return false
	}
	switch r.URL.Path {
	case "/api/health", "/api/particles", "/api/particles/feed":
		return true
	default:
		return false
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.RequestURI(), recorder.status, time.Since(started).Round(time.Millisecond))
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic %s %s: %v", r.Method, r.URL.Path, recovered)
				web.WriteError(w, apperror.Internal("Внутренняя ошибка сервера"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.written {
		r.status = status
		r.written = true
		r.ResponseWriter.WriteHeader(status)
	}
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	r.written = true
	return r.ResponseWriter.Write(data)
}

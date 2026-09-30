package httpapi

import (
	"log"
	"net/http"
	"time"

	"particleapp/internal/apperror"
	"particleapp/internal/domain/particle"
	"particleapp/internal/domain/user"
	"particleapp/internal/filestorage"
	"particleapp/internal/storage/postgres"
	"particleapp/internal/web"
)

type Dependencies struct {
	Particles particle.Domain
	Users     user.Domain
	Store     *postgres.Store
	Files     *filestorage.MinIO
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
	mux.Handle("/api/", middleware(api))
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

		payload := map[string]string{
			"api":    "/api",
			"bucket": deps.Files.Bucket(),
		}
		web.WriteJSON(w, http.StatusOK, payload)
	}
}

func middleware(next http.Handler) http.Handler {
	return recoverPanic(logRequests(withCORS(next)))
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

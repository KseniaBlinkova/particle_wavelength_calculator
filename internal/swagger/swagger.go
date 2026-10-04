package swagger

import (
	_ "embed"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

//go:embed openapi.json
var specification []byte

func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("GET /swagger/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(specification)
	})
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/openapi.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.PersistAuthorization(true),
	))
}

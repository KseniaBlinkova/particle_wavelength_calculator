package particlehandler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParticleMuxRoutes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		contains   string
	}{
		{name: "root redirects", path: "/", wantStatus: http.StatusSeeOther},
		{name: "feed", path: "/feed", wantStatus: http.StatusOK, contains: "Электрон"},
		{name: "feed collapse control", path: "/feed", wantStatus: http.StatusOK, contains: "Свернуть"},
		{name: "feed full description", path: "/feed", wantStatus: http.StatusOK, contains: "стабильная элементарная частица"},
		{name: "published particle", path: "/feed?id=2", wantStatus: http.StatusOK, contains: "Протон"},
		{name: "draft is hidden", path: "/feed?id=3", wantStatus: http.StatusNotFound},
		{name: "deleted is hidden", path: "/feed?id=4", wantStatus: http.StatusNotFound},
		{name: "invalid id", path: "/feed?id=abc", wantStatus: http.StatusBadRequest},
		{name: "editable draft mass", path: "/draft", wantStatus: http.StatusOK, contains: `name="mass"`},
		{name: "draft description", path: "/draft", wantStatus: http.StatusOK, contains: `name="description"`},
		{name: "readonly draft status", path: "/draft", wantStatus: http.StatusOK, contains: `value="Черновик" readonly`},
		{name: "disabled save button", path: "/draft", wantStatus: http.StatusOK, contains: `class="save-btn" disabled`},
		{name: "cards use range slider", path: "/cards", wantStatus: http.StatusOK, contains: `type="range"`},
		{name: "valid filter", path: "/cards?filter=100", wantStatus: http.StatusOK, contains: "Нейтрон"},
		{name: "invalid filter", path: "/cards?filter=NaN", wantStatus: http.StatusBadRequest},
		{name: "particle styles", path: "/static/css/particles.css", wantStatus: http.StatusOK, contains: ".phone-frame"},
	}

	mux := NewParticleMux()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.contains != "" && !strings.Contains(response.Body.String(), tt.contains) {
				t.Fatalf("body does not contain %q", tt.contains)
			}
		})
	}
}

func TestParticleMuxRejectsUnsupportedMethod(t *testing.T) {
	for _, path := range []string{"/feed", "/draft", "/cards"} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		response := httptest.NewRecorder()

		NewParticleMux().ServeHTTP(response, request)

		if response.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: status = %d, want %d", path, response.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestParticleCardsPageShowsAtMostFourCards(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/cards", nil)
	response := httptest.NewRecorder()

	NewParticleMux().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if count := strings.Count(response.Body.String(), `class="card-item"`); count != 4 {
		t.Fatalf("card count = %d, want 4", count)
	}
	if !strings.Contains(response.Body.String(), "Позитрон") {
		t.Fatal("default cards do not contain positron")
	}
}

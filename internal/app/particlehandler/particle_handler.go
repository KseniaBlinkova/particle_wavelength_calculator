package particlehandler

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	particleassets "particleapp"
	"particleapp/internal/particlerepository"
)

var particleTemplates = template.Must(template.ParseFS(particleassets.ParticleFiles, "resources/templates/*.html"))

const (
	ParticleMediaBaseURL = "http://localhost:9000/de-broglie/"
	DefaultImageKey      = "default.jpg" // ← ключ в MinIO
	DefaultVideoKey      = "default.mp4" // ← ключ в MinIO
)

func NewParticleMux() *http.ServeMux {
	mux := http.NewServeMux()

	staticFS, err := fs.Sub(particleassets.ParticleFiles, "resources/static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// GET
	mux.HandleFunc("GET /feed", ParticleFeedHandler)
	mux.HandleFunc("GET /draft", ParticleDraftHandler)
	mux.HandleFunc("GET /cards", ParticleCardsHandler)

	// POST
	mux.HandleFunc("POST /draft", ParticleCreateHandler)
	mux.HandleFunc("POST /publish", ParticlePublishHandler)
	mux.HandleFunc("POST /delete", ParticleDeleteHandler)

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/feed", http.StatusSeeOther)
	})
	return mux
}

func renderParticleTemplate(w http.ResponseWriter, particleTemplateName string, particlePageData any) {
	var renderedParticlePage bytes.Buffer
	if err := particleTemplates.ExecuteTemplate(&renderedParticlePage, particleTemplateName, particlePageData); err != nil {
		log.Printf("render particle template %s: %v", particleTemplateName, err)
		http.Error(w, "Не удалось сформировать страницу", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := renderedParticlePage.WriteTo(w); err != nil {
		log.Printf("write particle template %s: %v", particleTemplateName, err)
	}
}

// mediaURL — возвращает полный URL файла из MinIO.
// Если ключ пустой — возвращает URL файла по умолчанию (тоже из MinIO).
func mediaURL(key, baseURL, defaultKey string) string {
	if key == "" {
		return baseURL + defaultKey
	}
	return baseURL + key
}

// GET /feed

func ParticleFeedHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	next := r.URL.Query().Get("next") == "true"

	var particle *particlerepository.Particle

	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "Некорректный идентификатор частицы", http.StatusBadRequest)
			return
		}
		particle = particlerepository.GetPublishedParticleByID(id)
		if particle == nil {
			http.Error(w, "Частица не найдена", http.StatusNotFound)
			return
		}

		if next {
			published := particlerepository.GetPublishedParticles()
			for i, p := range published {
				if p.ID == particle.ID && i+1 < len(published) {
					nextP := published[i+1]
					particle = &nextP
					break
				}
			}
		}
	}

	if particle == nil {
		published := particlerepository.GetPublishedParticles()
		if len(published) > 0 {
			particle = &published[0]
		}
	}

	if particle == nil {
		http.Error(w, "Нет частиц", http.StatusNotFound)
		return
	}

	published := particlerepository.GetPublishedParticles()
	hasNext := false
	for i, p := range published {
		if p.ID == particle.ID && i+1 < len(published) {
			hasNext = true
			break
		}
	}

	particleFeedData := struct {
		Particle   particlerepository.Particle
		ImageURL   string
		VideoURL   string
		LikesCount int
		HasNext    bool
	}{
		Particle:   *particle,
		ImageURL:   mediaURL(particle.ImageKey, ParticleMediaBaseURL, DefaultImageKey),
		VideoURL:   mediaURL(particle.VideoKey, ParticleMediaBaseURL, DefaultVideoKey),
		LikesCount: particlerepository.GetParticleLikesCount(particle.ID),
		HasNext:    hasNext,
	}

	renderParticleTemplate(w, "particle_feed.html", particleFeedData)
}

//  GET /draft

func ParticleDraftHandler(w http.ResponseWriter, r *http.Request) {
	particle := particlerepository.GetDraftParticle()

	particleDraftData := struct {
		Particle particlerepository.Particle
		ImageURL string
		VideoURL string
		HasDraft bool
	}{
		ImageURL: ParticleMediaBaseURL + DefaultImageKey,
		VideoURL: ParticleMediaBaseURL + DefaultVideoKey,
		HasDraft: particle != nil,
	}

	if particle != nil {
		particleDraftData.Particle = *particle
		particleDraftData.ImageURL = mediaURL(particle.ImageKey, ParticleMediaBaseURL, DefaultImageKey)
		particleDraftData.VideoURL = mediaURL(particle.VideoKey, ParticleMediaBaseURL, DefaultVideoKey)
	}

	renderParticleTemplate(w, "particle_draft.html", particleDraftData)
}

// GET /cards

func ParticleCardsHandler(w http.ResponseWriter, r *http.Request) {
	const filterSteps = 100

	filterPosition := 0
	if filter := r.URL.Query().Get("filter"); filter != "" {
		position, err := strconv.Atoi(filter)
		if err != nil || position < 0 || position > filterSteps {
			http.Error(w, "Некорректное положение ползунка", http.StatusBadRequest)
			return
		}
		filterPosition = position
	}

	published := particlerepository.GetPublishedParticles()
	minAvailableMass, maxAvailableMass := particleMassRange(published)
	particles := published
	filterLabel := "0 кг"

	if filterPosition > 0 && minAvailableMass > 0 && maxAvailableMass >= minAvailableMass {
		fraction := float64(filterPosition-1) / float64(filterSteps-1)
		minMass := minAvailableMass * math.Pow(maxAvailableMass/minAvailableMass, fraction)
		particles = particlerepository.FilterParticlesByMass(minMass)
		filterLabel = fmt.Sprintf("%.3e кг", minMass)
	}
	sort.SliceStable(particles, func(i, j int) bool {
		return particles[i].Mass < particles[j].Mass
	})

	type ParticleCardData struct {
		Particle   particlerepository.Particle
		ImageURL   string
		LikesCount int
	}

	var particleCards []ParticleCardData
	for _, p := range particles {
		particleCards = append(particleCards, ParticleCardData{
			Particle:   p,
			ImageURL:   mediaURL(p.ImageKey, ParticleMediaBaseURL, DefaultImageKey),
			LikesCount: particlerepository.GetParticleLikesCount(p.ID),
		})
	}

	particleCardsPageData := struct {
		Cards       []ParticleCardData
		Filter      int
		FilterLabel string
		MinMass     float64
		MaxMass     float64
		FilterSteps int
	}{
		Cards:       particleCards,
		Filter:      filterPosition,
		FilterLabel: filterLabel,
		MinMass:     minAvailableMass,
		MaxMass:     maxAvailableMass,
		FilterSteps: filterSteps,
	}

	renderParticleTemplate(w, "particle_cards.html", particleCardsPageData)
}

func particleMassRange(particles []particlerepository.Particle) (float64, float64) {
	if len(particles) == 0 {
		return 0, 0
	}

	minMass := math.Inf(1)
	maxMass := 0.0
	for _, particle := range particles {
		if particle.Mass > 0 && particle.Mass < minMass {
			minMass = particle.Mass
		}
		if particle.Mass > maxMass {
			maxMass = particle.Mass
		}
	}
	if math.IsInf(minMass, 1) {
		return 0, maxMass
	}
	return minMass, maxMass
}

//  POST /draft

func ParticleCreateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", http.StatusBadRequest)
		return
	}

	existingDraft := particlerepository.GetDraftParticle()
	if existingDraft != nil {
		http.Redirect(w, r, "/draft", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Название обязательно", http.StatusBadRequest)
		return
	}

	mass, _ := strconv.ParseFloat(r.FormValue("mass"), 64)
	spin := r.FormValue("spin")
	wavelength, _ := strconv.ParseFloat(r.FormValue("wavelength"), 64)

	particle := &particlerepository.Particle{
		Name:       name,
		Mass:       mass,
		Spin:       spin,
		Wavelength: wavelength,
		Status:     particlerepository.StatusDraft,
		CreatorID:  1,
	}

	if err := particlerepository.CreateParticle(particle); err != nil {
		http.Error(w, "Ошибка создания частицы", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/draft", http.StatusSeeOther)
}

//  POST /publish

func ParticlePublishHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	mass, _ := strconv.ParseFloat(r.FormValue("mass"), 64)
	wavelength, _ := strconv.ParseFloat(r.FormValue("wavelength"), 64)

	updates := map[string]interface{}{
		"name":             r.FormValue("name"),
		"mass":             mass,
		"spin":             r.FormValue("spin"),
		"wavelength":       wavelength,
		"full_description": r.FormValue("full_description"),
		"status":           particlerepository.StatusPublished,
	}

	if err := particlerepository.UpdateParticle(uint(id), updates); err != nil {
		http.Error(w, "Ошибка публикации", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/cards", http.StatusSeeOther)
}

//  POST /delete

func ParticleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	if err := particlerepository.DeleteParticleSQL(uint(id)); err != nil {
		http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/cards", http.StatusSeeOther)
}

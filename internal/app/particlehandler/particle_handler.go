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

const ParticleMediaBaseURL = "http://localhost:9000/de-broglie/"

func NewParticleMux() *http.ServeMux {
	mux := http.NewServeMux()

	staticFS, err := fs.Sub(particleassets.ParticleFiles, "resources/static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /feed", ParticleFeedHandler)
	mux.HandleFunc("GET /draft", ParticleDraftHandler)
	mux.HandleFunc("GET /cards", ParticleCardsHandler)
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

// ===================== /feed =====================

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
		ImageURL:   ParticleMediaBaseURL + particle.ImageKey,
		VideoURL:   ParticleMediaBaseURL + particle.VideoKey,
		LikesCount: particlerepository.GetParticleLikesCount(particle.Likes),
		HasNext:    hasNext,
	}

	renderParticleTemplate(w, "particle_feed.html", particleFeedData)
}

// ===================== /draft =====================

func ParticleDraftHandler(w http.ResponseWriter, r *http.Request) {
	particle := particlerepository.GetDraftParticle()
	if particle == nil {
		http.Error(w, "Нет черновика", http.StatusNotFound)
		return
	}

	particleDraftData := struct {
		Particle particlerepository.Particle
		ImageURL string
		VideoURL string
	}{
		Particle: *particle,
		ImageURL: ParticleMediaBaseURL + particle.ImageKey,
		VideoURL: ParticleMediaBaseURL + particle.VideoKey,
	}

	renderParticleTemplate(w, "particle_draft.html", particleDraftData)
}

// ===================== /cards =====================

func ParticleCardsHandler(w http.ResponseWriter, r *http.Request) {
	const filterSteps = 100
	const maxCards = 4

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
	for i, p := range particles {
		if i >= maxCards {
			break
		}
		particleCards = append(particleCards, ParticleCardData{
			Particle:   p,
			ImageURL:   ParticleMediaBaseURL + p.ImageKey,
			LikesCount: particlerepository.GetParticleLikesCount(p.Likes),
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

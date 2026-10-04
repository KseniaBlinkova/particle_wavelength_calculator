package particle

import (
	"io"
	"time"

	"particleapp/internal/models"
)

// ParticleResponse — сериализатор услуги для клиента. Системные поля
// (creator_id, image_key, video_key, completed_at) наружу
// не отдаются: клиент получает только публичные ссылки и вычисленные признаки.
type ParticleResponse struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	FullDescription string    `json:"full_description"`
	Mass            *float64  `json:"mass"`
	Spin            string    `json:"spin"`
	ImageURL        string    `json:"image_url"`
	VideoURL        string    `json:"video_url"`
	LikesCount      int       `json:"likes_count"`
	IsLiked         int       `json:"is_liked"`
	CreatedAt       time.Time `json:"created_at"`
}

type MassRange struct {
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}

// ListResponse — GET /api/particles: только опубликованные услуги с фильтрацией.
type ListResponse struct {
	Items     []ParticleResponse `json:"items"`
	Count     int                `json:"count"`
	MassRange MassRange          `json:"mass_range"`
}

// FeedResponse — GET /api/particles/feed: одна услуга.
type FeedResponse struct {
	Particle ParticleResponse `json:"particle"`
}

// LikeResponse — POST /api/particles/{id}/like.
type LikeResponse struct {
	ParticleID uint `json:"particle_id"`
	Liked      int  `json:"liked"`
	LikesCount int  `json:"likes_count"`
}

// UploadedFile — файл, переданный клиентом как файл, а не как URL.
type UploadedFile struct {
	Reader       io.Reader
	OriginalName string
	Size         int64
	ContentType  string
	closer       io.Closer
}

func (f *UploadedFile) Close() error {
	if f == nil || f.closer == nil {
		return nil
	}
	return f.closer.Close()
}

// CreateRequest — POST /api/particles (multipart/form-data). В теле запроса
// нет и не может быть идентификатора, статуса, создателя и дат.
type CreateRequest struct {
	Name            string
	Mass            *float64
	Spin            string
	FullDescription string
	Image           *UploadedFile
	Video           *UploadedFile
}

// PublishRequest — PUT /api/particles/{id}/publish: редактируемые поля.
// Отсутствующее поле не меняется, status приходит только с бэкенда.
type PublishRequest struct {
	Mass            *float64 `json:"mass"`
	Spin            *string  `json:"spin"`
	FullDescription *string  `json:"full_description"`
}

// LikeRequest — POST /api/particles/{id}/like: 1 ставит лайк, 0 отменяет.
type LikeRequest struct {
	Liked int `json:"liked"`
}

func newParticleResponse(particle models.Particle, stat models.LikeStat, files Files) ParticleResponse {
	return ParticleResponse{
		ID:              particle.ID,
		Name:            particle.Name,
		FullDescription: particle.FullDescription,
		Mass:            particle.Mass,
		Spin:            particle.Spin,
		ImageURL:        files.URL(particle.ImageKey, defaultImageKey),
		VideoURL:        files.URL(particle.VideoKey, defaultVideoKey),
		LikesCount:      stat.Count,
		IsLiked:         flag(stat.LikedByMe),
		CreatedAt:       particle.CreatedAt,
	}
}

func flag(value bool) int {
	if value {
		return 1
	}
	return 0
}

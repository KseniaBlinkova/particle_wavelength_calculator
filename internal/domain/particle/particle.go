package particle

import (
	"context"
	"io"
	"time"

	"particleapp/internal/models"
)

// Filter — параметры фильтрации списка услуг (GET /api/particles).
type Filter struct {
	Name    string
	MinMass *float64
}

// Repository — порт хранения, который реализует internal/storage/postgres.
type Repository interface {
	ListPublished(ctx context.Context, name string, minMass *float64) ([]models.Particle, error)
	FirstPublished(ctx context.Context) (*models.Particle, error)
	GetNextPublished(ctx context.Context, id uint) (*models.Particle, error)
	GetPublishedByID(ctx context.Context, id uint) (*models.Particle, error)
	GetByID(ctx context.Context, id uint) (*models.Particle, error)
	GetDraftByCreator(ctx context.Context, creatorID uint) (*models.Particle, error)
	PublishedMassRange(ctx context.Context) (minMass, maxMass *float64, err error)
	Create(ctx context.Context, particle *models.Particle) error
	UpdateContent(ctx context.Context, id uint, mass *float64, spin, fullDescription string) error
	UpdateStatus(ctx context.Context, id, ownerID uint, from, to models.ParticleStatus, at time.Time) (bool, error)
	LikeStats(ctx context.Context, particleIDs []uint, viewerID uint) (map[uint]models.LikeStat, error)
	SetLike(ctx context.Context, userID, particleID uint, liked bool) (int, error)
}

// Files — порт файлового хранилища, который реализует internal/filestorage (MinIO).
type Files interface {
	Upload(ctx context.Context, key string, file io.Reader, size int64, contentType string) error
	URL(key, fallbackKey string) string
}

// Domain — бизнес-логика домена «услуга»: один метод на каждый URL API.
type Domain interface {
	List(ctx context.Context, filter Filter) (ListResponse, error)
	Feed(ctx context.Context, id *uint, next bool) (FeedResponse, error)
	Draft(ctx context.Context) (*ParticleResponse, error)
	Create(ctx context.Context, request CreateRequest) (*ParticleResponse, error)
	Publish(ctx context.Context, id uint, request PublishRequest) (*ParticleResponse, error)
	Delete(ctx context.Context, id uint) error
	Like(ctx context.Context, id uint, liked int) (LikeResponse, error)
}

type Usecase struct {
	repo  Repository
	files Files
	now   func() time.Time
}

func New(repo Repository, files Files) *Usecase {
	return &Usecase{repo: repo, files: files, now: time.Now}
}

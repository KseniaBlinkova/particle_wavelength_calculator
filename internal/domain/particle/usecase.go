package particle

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"particleapp/internal/apperror"
	"particleapp/internal/auth"
	"particleapp/internal/models"
	"particleapp/internal/storage"
)

const (
	defaultImageKey = "default.jpg"
	defaultVideoKey = "default.mp4"

	maxNameLength  = 100
	maxSpinLength  = 10
	maxImageSize   = 10 << 20
	maxVideoSize   = 50 << 20
	allowedImage   = "image/jpeg,image/png,image/webp"
	allowedVideo   = "video/mp4,video/webm"
	draftExistCode = "draft_already_exists"
)

// List — GET /api/particles. Только опубликованные услуги с фильтрацией
// по минимальной массе и поиском по названию. Записи в статусе deleted
// и черновики в выборку не попадают.
func (u *Usecase) List(ctx context.Context, filter Filter) (ListResponse, error) {
	particles, err := u.repo.ListPublished(ctx, filter.Name, filter.MinMass)
	if err != nil {
		return ListResponse{}, apperror.Internal("Не удалось получить список услуг")
	}

	minMass, maxMass, err := u.repo.PublishedMassRange(ctx)
	if err != nil {
		return ListResponse{}, apperror.Internal("Не удалось получить диапазон масс")
	}

	items, err := u.responses(ctx, particles)
	if err != nil {
		return ListResponse{}, err
	}

	return ListResponse{Items: items, Count: len(items), MassRange: MassRange{Min: minMass, Max: maxMass}}, nil
}

// Feed — GET /api/particles/feed. Без id отдаёт первую опубликованную услугу,
// с id — указанную, с next=true — следующую после неё.
func (u *Usecase) Feed(ctx context.Context, id *uint, next bool) (FeedResponse, error) {
	var (
		particle *models.Particle
		err      error
	)

	switch {
	case id == nil:
		particle, err = u.repo.FirstPublished(ctx)
		if err != nil {
			return FeedResponse{}, notFoundErr(err, "feed_is_empty", "В ленте нет опубликованных услуг")
		}
	case next:
		current, currentErr := u.repo.GetPublishedByID(ctx, *id)
		if currentErr != nil {
			return FeedResponse{}, notFoundErr(currentErr, "particle_not_found", "Услуга не найдена")
		}
		particle, err = u.repo.GetNextPublished(ctx, current.ID)
		if err != nil {
			return FeedResponse{}, notFoundErr(err, "feed_is_empty", "Это последняя услуга в ленте")
		}
	default:
		particle, err = u.repo.GetPublishedByID(ctx, *id)
		if err != nil {
			return FeedResponse{}, notFoundErr(err, "particle_not_found", "Услуга не найдена")
		}
	}

	response, err := u.responses(ctx, []models.Particle{*particle})
	if err != nil {
		return FeedResponse{}, err
	}

	return FeedResponse{Particle: response[0]}, nil
}

// Draft — GET /api/particles/draft. У пользователя не более одной записи
// в статусе draft, идентификатор с клиента не передаётся.
func (u *Usecase) Draft(ctx context.Context) (*ParticleResponse, error) {
	creatorID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	draft, err := u.repo.GetDraftByCreator(ctx, creatorID)
	if err != nil {
		return nil, notFoundErr(err, "draft_not_found", "Черновик не найден")
	}

	items, err := u.responses(ctx, []models.Particle{*draft})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

// Create — POST /api/particles. Создаёт черновик, файлы кладутся в MinIO,
// а их имена сохраняются в полях image_key и video_key. Создатель,
// статус и даты вычисляются на бэкенде.
func (u *Usecase) Create(ctx context.Context, request CreateRequest) (*ParticleResponse, error) {
	creatorID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := u.repo.GetDraftByCreator(ctx, creatorID); err == nil {
		return nil, apperror.Conflict(draftExistCode, "У пользователя уже есть черновик")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, apperror.Internal("Не удалось проверить черновик")
	}

	name := strings.TrimSpace(request.Name)
	if err := validateContent(name, request.Mass, request.Spin); err != nil {
		return nil, err
	}

	now := u.now()
	particle := models.Particle{
		Name:            name,
		FullDescription: strings.TrimSpace(request.FullDescription),
		Mass:            request.Mass,
		Spin:            strings.TrimSpace(request.Spin),
		Status:          models.StatusDraft,
		CreatorID:       creatorID,
		CreatedAt:       now,
	}

	if request.Image != nil {
		key, err := u.upload(ctx, "image", request.Image, maxImageSize, allowedImage, now)
		if err != nil {
			return nil, err
		}
		particle.ImageKey = key
	}

	if request.Video != nil {
		key, err := u.upload(ctx, "video", request.Video, maxVideoSize, allowedVideo, now)
		if err != nil {
			return nil, err
		}
		particle.VideoKey = key
	}

	if err := u.repo.Create(ctx, &particle); err != nil {
		return nil, apperror.Internal("Не удалось создать услугу")
	}

	items, err := u.responses(ctx, []models.Particle{particle})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

// Publish — PUT /api/particles/{id}/publish. Смена статуса draft -> published
// с проставлением даты формирования. Возврат в черновик не предусмотрен.
func (u *Usecase) Publish(ctx context.Context, id uint, request PublishRequest) (*ParticleResponse, error) {
	particle, err := u.ownedParticle(ctx, id)
	if err != nil {
		return nil, err
	}
	if particle.Status == models.StatusDeleted {
		return nil, apperror.NotFound("particle_not_found", "Услуга не найдена")
	}
	if particle.Status != models.StatusDraft {
		return nil, apperror.Conflict("invalid_status", "Опубликовать можно только черновик")
	}
	if !particle.Status.CanTransitionTo(models.StatusPublished) {
		return nil, apperror.Conflict("invalid_status", "Такой переход статуса запрещён")
	}

	if request.Mass != nil {
		mass := *request.Mass
		particle.Mass = &mass
	}
	if request.Spin != nil {
		particle.Spin = strings.TrimSpace(*request.Spin)
	}
	if request.FullDescription != nil {
		particle.FullDescription = strings.TrimSpace(*request.FullDescription)
	}
	if err := validateContent(particle.Name, particle.Mass, particle.Spin); err != nil {
		return nil, err
	}

	now := u.now()
	if err := u.repo.UpdateContent(ctx, particle.ID, particle.Mass, particle.Spin, particle.FullDescription); err != nil {
		return nil, apperror.Internal("Не удалось сохранить услугу")
	}

	updated, err := u.repo.UpdateStatus(ctx, particle.ID, particle.CreatorID, models.StatusDraft, models.StatusPublished, now)
	if err != nil {
		return nil, apperror.Internal("Не удалось опубликовать услугу")
	}
	if !updated {
		return nil, apperror.Conflict("invalid_status", "Услуга уже не находится в статусе черновика")
	}

	published, err := u.repo.GetPublishedByID(ctx, particle.ID)
	if err != nil {
		return nil, apperror.Internal("Не удалось получить опубликованную услугу")
	}

	items, err := u.responses(ctx, []models.Particle{*published})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

// Delete — DELETE /api/particles/{id}. Только soft delete и только для
// услуги текущего пользователя: статус deleted и дата завершения.
func (u *Usecase) Delete(ctx context.Context, id uint) error {
	particle, err := u.ownedParticle(ctx, id)
	if err != nil {
		return err
	}
	if particle.Status == models.StatusDeleted {
		return apperror.NotFound("particle_not_found", "Услуга не найдена")
	}
	if !particle.Status.CanTransitionTo(models.StatusDeleted) {
		return apperror.Conflict("invalid_status", "Такой переход статуса запрещён")
	}

	deleted, err := u.repo.UpdateStatus(ctx, particle.ID, particle.CreatorID, particle.Status, models.StatusDeleted, u.now())
	if err != nil {
		return apperror.Internal("Не удалось удалить услугу")
	}
	if !deleted {
		return apperror.Conflict("invalid_status", "Услуга уже не находится в статусе "+string(particle.Status))
	}
	return nil
}

// Like — POST /api/particles/{id}/like. liked=1 ставит лайк от текущего
// пользователя, liked=0 отменяет его.
func (u *Usecase) Like(ctx context.Context, id uint, liked int) (LikeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return LikeResponse{}, err
	}
	if liked != 0 && liked != 1 {
		return LikeResponse{}, apperror.Validation("Поле liked принимает только 0 или 1")
	}

	particle, err := u.repo.GetPublishedByID(ctx, id)
	if err != nil {
		return LikeResponse{}, notFoundErr(err, "particle_not_found", "Услуга не найдена")
	}

	count, err := u.repo.SetLike(ctx, userID, particle.ID, liked == 1)
	if err != nil {
		return LikeResponse{}, apperror.Internal("Не удалось сохранить лайк")
	}

	return LikeResponse{ParticleID: particle.ID, Liked: liked, LikesCount: count}, nil
}

func (u *Usecase) responses(ctx context.Context, particles []models.Particle) ([]ParticleResponse, error) {
	viewerID, _ := auth.UserID(ctx)

	ids := make([]uint, 0, len(particles))
	for _, particle := range particles {
		ids = append(ids, particle.ID)
	}

	stats, err := u.repo.LikeStats(ctx, ids, viewerID)
	if err != nil {
		return nil, apperror.Internal("Не удалось получить лайки")
	}

	items := make([]ParticleResponse, 0, len(particles))
	for _, particle := range particles {
		items = append(items, newParticleResponse(particle, stats[particle.ID], u.files, viewerID))
	}
	return items, nil
}

// ownedParticle — запись существует и принадлежит текущему пользователю.
func (u *Usecase) ownedParticle(ctx context.Context, id uint) (*models.Particle, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	particle, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundErr(err, "particle_not_found", "Услуга не найдена")
	}
	if particle.CreatorID != userID {
		return nil, apperror.Forbidden("Изменять можно только свои услуги")
	}
	return particle, nil
}

func currentUserID(ctx context.Context) (uint, error) {
	userID, ok := auth.UserID(ctx)
	if !ok {
		return 0, apperror.Unauthorized("Требуется авторизация")
	}
	return userID, nil
}

func (u *Usecase) upload(ctx context.Context, kind string, file *UploadedFile, maxSize int64, allowedTypes string, at time.Time) (string, error) {
	if file.Size <= 0 {
		return "", apperror.Validation("Файл " + kind + " пуст")
	}
	if file.Size > maxSize {
		return "", apperror.PayloadTooLarge("file_too_large", fmt.Sprintf("Файл %s больше допустимых %d МБ", kind, maxSize>>20))
	}
	if !contains(allowedTypes, file.ContentType) {
		return "", apperror.UnsupportedMediaType(fmt.Sprintf("Недопустимый тип файла %s, ожидается: %s", kind, allowedTypes))
	}

	key := newFileKey(kind, file.ContentType, file.OriginalName, at)
	if err := u.files.Upload(ctx, key, file.Reader, file.Size, file.ContentType); err != nil {
		return "", apperror.Internal("Не удалось загрузить файл " + kind)
	}
	return key, nil
}

func validateContent(name string, mass *float64, spin string) error {
	if name == "" {
		return apperror.Validation("Название обязательно")
	}
	if len([]rune(name)) > maxNameLength {
		return apperror.Validation("Название длиннее 100 символов")
	}
	if len([]rune(spin)) > maxSpinLength {
		return apperror.Validation("Спин длиннее 10 символов")
	}
	if mass != nil && (math.IsNaN(*mass) || math.IsInf(*mass, 0) || *mass < 0) {
		return apperror.Validation("Масса должна быть конечным неотрицательным числом")
	}
	return nil
}

func notFoundErr(err error, code, message string) error {
	if errors.Is(err, storage.ErrNotFound) {
		return apperror.NotFound(code, message)
	}
	return apperror.Internal("Ошибка обращения к базе данных")
}

func contains(list, value string) bool {
	for _, item := range strings.Split(list, ",") {
		if item == value {
			return true
		}
	}
	return false
}

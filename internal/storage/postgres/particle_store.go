package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"particleapp/internal/models"
	"particleapp/internal/storage"
)

func (s *Store) ListPublished(ctx context.Context, name string, minMass *float64) ([]models.Particle, error) {
	query := s.db.WithContext(ctx).
		Where("particles.status = ?", models.StatusPublished)

	if minMass != nil {
		query = query.Where("particles.mass >= ?", *minMass)
	}

	if name != "" {
		query = query.Where("particles.name ILIKE ?", "%"+escapeLike(name)+"%")
	}

	var particles []models.Particle
	if err := query.Order("particles.id ASC").Find(&particles).Error; err != nil {
		return nil, err
	}
	return particles, nil
}

func (s *Store) FirstPublished(ctx context.Context) (*models.Particle, error) {
	return s.oneParticle(ctx, s.db.WithContext(ctx).
		Where("status = ?", models.StatusPublished).
		Order("id ASC"))
}

func (s *Store) GetNextPublished(ctx context.Context, id uint) (*models.Particle, error) {
	return s.oneParticle(ctx, s.db.WithContext(ctx).
		Where("status = ? AND id > ?", models.StatusPublished, id).
		Order("id ASC"))
}

func (s *Store) GetPublishedByID(ctx context.Context, id uint) (*models.Particle, error) {
	return s.oneParticle(ctx, s.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, models.StatusPublished))
}

func (s *Store) GetByID(ctx context.Context, id uint) (*models.Particle, error) {
	return s.oneParticle(ctx, s.db.WithContext(ctx).Where("id = ?", id))
}

func (s *Store) GetDraftByCreator(ctx context.Context, creatorID uint) (*models.Particle, error) {
	return s.oneParticle(ctx, s.db.WithContext(ctx).
		Where("status = ? AND creator_id = ?", models.StatusDraft, creatorID).
		Order("id DESC"))
}

func (s *Store) PublishedMassRange(ctx context.Context) (*float64, *float64, error) {
	var massRange struct {
		MinMass *float64
		MaxMass *float64
	}
	err := s.db.WithContext(ctx).Model(&models.Particle{}).
		Select("MIN(mass) AS min_mass, MAX(mass) AS max_mass").
		Where("status = ?", models.StatusPublished).
		Scan(&massRange).Error
	if err != nil {
		return nil, nil, err
	}
	return massRange.MinMass, massRange.MaxMass, nil
}

func (s *Store) Create(ctx context.Context, particle *models.Particle) error {
	return s.db.WithContext(ctx).Create(particle).Error
}

func (s *Store) UpdateContent(ctx context.Context, id uint, mass *float64, spin, fullDescription string) error {
	return s.db.WithContext(ctx).Model(&models.Particle{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"mass":             mass,
			"spin":             spin,
			"full_description": fullDescription,
		}).Error
}

// UpdateStatus — условный переход статуса: строка обновляется только если
// она ещё находится в статусе from и принадлежит владельцу ownerID.
// Это делает смену статуса атомарной и не даёт опубликовать запись повторно.
func (s *Store) UpdateStatus(ctx context.Context, id, ownerID uint, from, to models.ParticleStatus, at time.Time) (bool, error) {
	updates := map[string]any{"status": to}
	switch to {
	case models.StatusPublished:
		updates["formed_at"] = at
	case models.StatusDeleted:
		updates["completed_at"] = at
	}

	result := s.db.WithContext(ctx).Model(&models.Particle{}).
		Where("id = ? AND creator_id = ? AND status = ?", id, ownerID, from).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (s *Store) LikeStats(ctx context.Context, particleIDs []uint, viewerID uint) (map[uint]models.LikeStat, error) {
	stats := make(map[uint]models.LikeStat, len(particleIDs))
	for _, id := range particleIDs {
		stats[id] = models.LikeStat{ParticleID: id}
	}
	if len(particleIDs) == 0 {
		return stats, nil
	}

	var likes []models.Like
	if err := s.db.WithContext(ctx).Where("particle_id IN ?", particleIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, like := range likes {
		stat := stats[like.ParticleID]
		stat.Count++
		stats[like.ParticleID] = stat
	}

	var mine []uint
	err := s.db.WithContext(ctx).Model(&models.Like{}).
		Where("particle_id IN ? AND user_id = ?", particleIDs, viewerID).
		Distinct().
		Pluck("particle_id", &mine).Error
	if err != nil {
		return nil, err
	}
	for _, id := range mine {
		stat := stats[id]
		stat.LikedByMe = true
		stats[id] = stat
	}

	return stats, nil
}

// SetLike — ставит или снимает лайк от текущего пользователя в транзакции.
// Перед созданием удаляет возможные дубликаты, поэтому повторный запрос
// с liked=1 не создаёт вторую запись.
func (s *Store) SetLike(ctx context.Context, userID, particleID uint, liked bool) (int, error) {
	var count int64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND particle_id = ?", userID, particleID).
			Delete(&models.Like{}).Error; err != nil {
			return err
		}

		if liked {
			like := models.Like{UserID: userID, ParticleID: particleID, CreatedAt: time.Now()}
			if err := tx.Create(&like).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.Like{}).
			Where("particle_id = ?", particleID).
			Count(&count).Error
	})
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
func (s *Store) oneParticle(ctx context.Context, query *gorm.DB) (*models.Particle, error) {
	var particle models.Particle
	if err := query.First(&particle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &particle, nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

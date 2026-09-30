package models

import "time"

type ParticleStatus string

const (
	StatusDraft     ParticleStatus = "draft"
	StatusPublished ParticleStatus = "published"
	StatusDeleted   ParticleStatus = "deleted"
)

// CanTransitionTo — таблица допустимых переходов статуса.
// Возврат в черновик не предусмотрен: у создателя два разных метода
// (публикация и удаление), поэтому published -> draft запрещён.
func (s ParticleStatus) CanTransitionTo(next ParticleStatus) bool {
	switch s {
	case StatusDraft:
		return next == StatusPublished || next == StatusDeleted
	case StatusPublished:
		return next == StatusDeleted
	default:
		return false
	}
}

type Particle struct {
	ID              uint           `gorm:"primaryKey;type:integer"`
	Name            string         `gorm:"size:100;not null"`
	FullDescription string         `gorm:"column:full_description;type:text"`
	Status          ParticleStatus `gorm:"size:20;not null;default:draft"`
	ImageKey        string         `gorm:"column:image_key;size:255"`
	VideoKey        string         `gorm:"column:video_key;size:255"`
	Mass            *float64       `gorm:"type:double precision"`
	Spin            string         `gorm:"size:10"`
	Wavelength      float64        `gorm:"type:double precision"`
	CreatorID       uint           `gorm:"type:integer;not null;uniqueIndex:idx_particles_one_draft_per_creator,where:status = 'draft'"`
	CreatedAt       time.Time      `gorm:"type:timestamp without time zone"`
	FormedAt        *time.Time     `gorm:"type:timestamp without time zone"`
	CompletedAt     *time.Time     `gorm:"type:timestamp without time zone"`

	Creator User `gorm:"foreignKey:CreatorID"`
}

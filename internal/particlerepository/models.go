package particlerepository

import "time"

const (
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusDeleted   = "deleted"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"size:50;not null"`
	Email    string `gorm:"size:100;not null;unique"`
}

type Particle struct {
	ID              uint   `gorm:"primaryKey"`
	Name            string `gorm:"size:100;not null"`
	FullDescription string `gorm:"column:full_description;type:text"`
	Status          string `gorm:"size:20;not null;default:draft"`
	ImageKey        string `gorm:"column:image_key;size:255"`
	VideoKey        string `gorm:"column:video_key;size:255"`
	Mass            float64
	Spin            string `gorm:"size:10"`
	CreatedAt       time.Time
	CreatorID       uint
	Creator         User `gorm:"foreignKey:CreatorID"`
	FormedAt        *time.Time
}

type Like struct {
	ID         uint     `gorm:"primaryKey"`
	UserID     uint     `gorm:"not null"`
	ParticleID uint     `gorm:"not null"`
	User       User     `gorm:"foreignKey:UserID"`
	Particle   Particle `gorm:"foreignKey:ParticleID"`
}

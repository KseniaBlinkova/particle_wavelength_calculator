package models

import "time"

type Like struct {
	ID         uint      `gorm:"primaryKey;type:integer"`
	UserID     uint      `gorm:"type:integer;not null;uniqueIndex:idx_likes_user_particle"`
	ParticleID uint      `gorm:"type:integer;not null;uniqueIndex:idx_likes_user_particle"`
	CreatedAt  time.Time `gorm:"type:timestamp without time zone"`

	User     User     `gorm:"foreignKey:UserID"`
	Particle Particle `gorm:"foreignKey:ParticleID"`
}

// LikeStat — агрегат лайков для карточки или объекта ленты.
type LikeStat struct {
	ParticleID uint
	Count      int
	LikedByMe  bool
}

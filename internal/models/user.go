package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey;type:integer"`
	Login     string    `gorm:"size:50;not null;unique"`
	Password  string    `gorm:"size:255;not null;default:''"`
	CreatedAt time.Time `gorm:"type:timestamp without time zone"`
}

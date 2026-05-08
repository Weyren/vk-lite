package models

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Email        string    `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	Name         string    `gorm:"size:255"`
	AvatarURL    string    `gorm:"size:512"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

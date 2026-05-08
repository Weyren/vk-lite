package models

import "time"

type Post struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	AuthorID  int64     `gorm:"not null;index"`
	Content   string    `gorm:"type:text"`
	MediaURL  string    `gorm:"size:512"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

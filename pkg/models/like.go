package models

type Like struct {
	UserID int64 `gorm:"primaryKey;autoIncrement:false"`
	PostID int64 `gorm:"primaryKey;autoIncrement:false;index"`
}

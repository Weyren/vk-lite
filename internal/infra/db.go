package infra

import (
	"github.com/Weyren/vk-lite/pkg/models"
	"github.com/Weyren/vk-lite/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(cfg *utils.Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
	if err != nil {
		panic("cannot connect to postgres: " + err.Error())
	}
	// Автомиграция только для учебных целей (в проде будем использовать миграции)
	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Like{}, &models.Subscription{}); err != nil {
		panic(err)
	}
	return db
}

package infra

import (
	"log"
	"time"

	"github.com/Weyren/vk-lite/pkg/models"
	"github.com/Weyren/vk-lite/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(cfg *utils.Config) *gorm.DB {
	var db *gorm.DB
	var err error

	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				err = sqlDB.Ping()
			}
		}
		if err == nil {
			break
		}

		log.Printf("postgres is not ready yet, attempt %d/30: %v", attempt, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		panic("cannot connect to postgres: " + err.Error())
	}

	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Like{}, &models.Subscription{}); err != nil {
		panic(err)
	}
	return db
}

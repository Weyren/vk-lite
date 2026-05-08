package models

type Subscription struct {
	SubscriberID int64 `gorm:"primaryKey;autoIncrement:false;index"`
	TargetID     int64 `gorm:"primaryKey;autoIncrement:false;index"`
}

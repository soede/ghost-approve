package models

import "time"

// HiddenReport – хранит, кто скрыл отчет
type HiddenReport struct {
	ID        int    `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	ApproveID int    `gorm:"index"`
	CreatedAt time.Time
}

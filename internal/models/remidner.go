package models

import "time"

type ApprovalReminder struct {
	ID         int       `gorm:"primaryKey"`
	ApprovalID int       `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime;not null"`
	Approval   Approval  `gorm:"foreignKey:ApprovalID"`
}

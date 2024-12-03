package models

import "time"

type User struct {
	ID             string `gorm:"primaryKey"`
	FirstName      string
	LastName       string
	Registered     bool
	MiddleSignTime uint
}

type RejectedUser struct {
	ApproveID int
	UserID    string
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
type ApprovedUser struct {
	ApproveID int
	UserID    string
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (RejectedUser) TableName() string {
	return "rejected_users"
}
func (ApprovedUser) TableName() string {
	return "approved_users"
}

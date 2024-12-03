package models

import (
	"github.com/lib/pq"
	"time"
)

type Approval struct {
	ID            int `gorm:"primaryKey"`
	AuthorID      string
	Author        User `gorm:"foreignKey:AuthorID"`
	Title         string
	Description   string
	Links         pq.StringArray `gorm:"type:text[]"`
	ConfirmTime   int
	Status        ApproveStatus `gorm:"type:varchar(50)"`
	Cancelable    bool          `gorm:"type:boolean"`
	Editable      bool          `gorm:"type:boolean"`
	StopOnReject  bool          `gorm:"type:boolean"`
	IsPrivate     bool          `gorm:"type:boolean"`
	CreatedAt     time.Time     `gorm:"autoCreateTime"`
	CompletedAt   time.Time     `gorm:"column:completed_at"`
	TotalComplete uint
	Participants  []User `gorm:"many2many:approval_users;"`

	File *File `gorm:"foreignKey:ApproveID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (Approval) TableName() string {
	return "approvals"
}

type ApproveStatus string

const (
	StatusPending           ApproveStatus = "Отравлено"
	StatusApproved          ApproveStatus = "Подтверждено"
	StatusRejected          ApproveStatus = "Отклонено"
	StatusCanceled          ApproveStatus = "Отменено"
	StatusExpired           ApproveStatus = "Истекло"
	StatusPartiallyApproved ApproveStatus = "Частично подтверждено"
)

package models

import "time"

type File struct {
	ID             int       `gorm:"primaryKey"`
	ApproveID      int       `gorm:"not null;index"`
	Approve        *Approval `gorm:"foreignKey:ApproveID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AuthorID       string    `gorm:"not null"`
	UploaderID     string    `gorm:"not null"`
	OriginalFileID string    `gorm:"not null"`
	FileID         string    `gorm:"not null"`
	Version        int       `gorm:"default:1"`
	UploadedAt     time.Time `gorm:"autoCreateTime"`
}

type FileHistory struct {
	ID         int       `gorm:"primaryKey"`
	ApproveID  int       `gorm:"not null"`
	FileID     string    `gorm:"not null"`
	UploaderID string    `gorm:"not null"`
	Version    int       `gorm:"not null"`
	UploadedAt time.Time `gorm:"autoCreateTime"`
}

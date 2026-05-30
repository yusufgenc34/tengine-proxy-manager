package model

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID *uint  `json:"user_id"`
	Action string `gorm:"not null" json:"action"`
	Detail string `json:"detail"`
	IP     string `json:"ip"`
}

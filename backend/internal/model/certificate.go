package model

import (
	"time"

	"gorm.io/gorm"
)

type Certificate struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Domain    string     `gorm:"not null" json:"domain"`
	Type      string     `gorm:"not null" json:"type"`
	ExpiresAt *time.Time `json:"expires_at"`
	CertPath  string     `json:"cert_path"`
	KeyPath   string     `json:"key_path"`

	ProxyHosts []ProxyHost `gorm:"foreignKey:CertificateID" json:"proxy_hosts,omitempty"`
}

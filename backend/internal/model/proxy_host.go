package model

import (
	"time"

	"gorm.io/gorm"
)

type ProxyHost struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Domain        string         `gorm:"uniqueIndex;not null" json:"domain"`
	ForwardHost   string         `gorm:"not null" json:"forward_host"`
	ForwardPort   int            `gorm:"not null" json:"forward_port"`
	ForwardScheme string         `gorm:"default:http" json:"forward_scheme"`
	SslEnabled    bool           `gorm:"default:false" json:"ssl_enabled"`
	HealthCheck   bool           `gorm:"default:false" json:"health_check"`
	LoadBalancing string         `json:"load_balancing"`
	Enabled       bool           `gorm:"default:true" json:"enabled"`

	CertificateID *uint        `json:"certificate_id"`
	Certificate   *Certificate `gorm:"foreignKey:CertificateID" json:"certificate,omitempty"`
	AccessListID  *uint        `json:"access_list_id"`
	AccessList    *AccessList  `gorm:"foreignKey:AccessListID" json:"access_list,omitempty"`
}

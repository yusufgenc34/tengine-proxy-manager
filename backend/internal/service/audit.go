package service

import (
	"tpm/internal/model"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Log(userID *uint, ip, action, detail string) {
	entry := model.AuditLog{
		UserID: userID,
		IP:     ip,
		Action: action,
		Detail: detail,
	}
	s.db.Create(&entry)
}

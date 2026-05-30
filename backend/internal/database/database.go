package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tpm/internal/model"
)

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	err = db.AutoMigrate(
		&model.ProxyHost{},
		&model.Certificate{},
		&model.AccessList{},
		&model.User{},
		&model.AuditLog{},
	)
	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	// Clean up old audit logs on startup (configurable via AUDIT_RETENTION_DAYS env, 0=disabled)
	retentionDays := 30
	if d := os.Getenv("AUDIT_RETENTION_DAYS"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			retentionDays = parsed
		}
	}
	if retentionDays > 0 {
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		result := db.Where("created_at < ?", cutoff).Delete(&model.AuditLog{})
		if result.RowsAffected > 0 {
			log.Printf("Cleaned up %d audit log(s) older than %d days", result.RowsAffected, retentionDays)
		}
	}

	return db, nil
}

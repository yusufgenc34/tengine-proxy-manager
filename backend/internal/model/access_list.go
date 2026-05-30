package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AccessList struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name  string `gorm:"not null" json:"name"`
	Rules string `gorm:"type:jsonb" json:"rules"`

	ProxyHosts []ProxyHost `gorm:"foreignKey:AccessListID" json:"proxy_hosts,omitempty"`
}

// AccessRule is a single allow/deny rule with IP or CIDR.
type AccessRule struct {
	IP     string `json:"ip"`
	Action string `json:"action"`
}

// ParsedRules returns the rules as parsed structs for template use.
func (a *AccessList) ParsedRules() []AccessRule {
	if a == nil || a.Rules == "" {
		return nil
	}
	var rules []AccessRule
	if err := json.Unmarshal([]byte(a.Rules), &rules); err != nil {
		return nil
	}
	return rules
}

package model

import "time"

// CloudflareSettings holds the Cloudflare IP whitelist configuration.
// Only one row (ID=1) is ever used.
type CloudflareSettings struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Enabled     bool      `gorm:"default:false" json:"enabled"`
	IPv4List    string    `gorm:"type:text" json:"-"`
	IPv6List    string    `gorm:"type:text" json:"-"`
	LastFetched time.Time `json:"last_fetched"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IPv4Count returns the number of cached IPv4 ranges.
func (c *CloudflareSettings) IPv4Count() int {
	if c.IPv4List == "" {
		return 0
	}
	count := 0
	for i := 0; i < len(c.IPv4List); i++ {
		if c.IPv4List[i] == '\n' {
			count++
		}
	}
	if len(c.IPv4List) > 0 && c.IPv4List[len(c.IPv4List)-1] != '\n' {
		count++
	}
	return count
}

// IPv6Count returns the number of cached IPv6 ranges.
func (c *CloudflareSettings) IPv6Count() int {
	if c.IPv6List == "" {
		return 0
	}
	count := 0
	for i := 0; i < len(c.IPv6List); i++ {
		if c.IPv6List[i] == '\n' {
			count++
		}
	}
	if len(c.IPv6List) > 0 && c.IPv6List[len(c.IPv6List)-1] != '\n' {
		count++
	}
	return count
}

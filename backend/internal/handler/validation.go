package handler

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidDomain(domain string) bool {
	if len(domain) > 255 || len(domain) == 0 {
		return false
	}
	return domainRegex.MatchString(domain)
}

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func isValidPort(port int) bool {
	return port >= 1 && port <= 65535
}

func isValidScheme(scheme string) bool {
	switch scheme {
	case "http", "https":
		return true
	}
	return false
}

func isValidLoadBalancing(lb string) bool {
	switch lb {
	case "", "least_conn", "consistent_hash":
		return true
	}
	return false
}

func sanitizeDomain(domain string) string {
	// Remove any path traversal attempts
	domain = strings.ReplaceAll(domain, "..", "")
	domain = strings.ReplaceAll(domain, "/", "")
	domain = strings.ReplaceAll(domain, "\\", "")
	domain = strings.ReplaceAll(domain, " ", "")
	return strings.ToLower(domain)
}

func isValidAccessListRules(rules string) bool {
	if rules == "" || rules == "[]" {
		return true
	}
	var parsed []struct {
		IP     string `json:"ip"`
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(rules), &parsed); err != nil {
		return false
	}
	for _, rule := range parsed {
		if rule.Action != "allow" && rule.Action != "deny" {
			return false
		}
		// Validate IP or CIDR
		if net.ParseIP(rule.IP) == nil {
			if _, _, err := net.ParseCIDR(rule.IP); err != nil {
				return false
			}
		}
	}
	return true
}

func isValidPassword(password string) bool {
	return len(password) >= 8
}

// ParseAccessListExpression converts a Cloudflare-style expression to JSON rules.
// Syntax:
//
//	allow 192.168.1.1
//	allow 10.0.0.0/8
//	deny 0.0.0.0/0
//
// Lines starting with # are comments. Empty lines are ignored.
func ParseAccessListExpression(expr string) (string, string) {
	var rules []struct {
		IP     string `json:"ip"`
		Action string `json:"action"`
	}
	lines := strings.Split(expr, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return "", fmt.Sprintf("Syntax error on line %d: expected 'allow|deny <ip/cidr>'", i+1)
		}
		action := strings.ToLower(parts[0])
		if action != "allow" && action != "deny" {
			return "", fmt.Sprintf("Invalid action on line %d: must be 'allow' or 'deny'", i+1)
		}
		ip := parts[1]
		if net.ParseIP(ip) == nil {
			if _, _, err := net.ParseCIDR(ip); err != nil {
				return "", fmt.Sprintf("Invalid IP/CIDR on line %d: %s", i+1, ip)
			}
		}
		rules = append(rules, struct {
			IP     string `json:"ip"`
			Action string `json:"action"`
		}{IP: ip, Action: action})
	}
	if len(rules) == 0 {
		return "", "At least one rule is required"
	}
	data, err := json.Marshal(rules)
	if err != nil {
		return "", "Failed to encode rules"
	}
	return string(data), ""
}

// RulesToExpression converts JSON rules to the expression format.
func RulesToExpression(rulesJSON string) string {
	var rules []struct {
		IP     string `json:"ip"`
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil || len(rules) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, r := range rules {
		buf.WriteString(r.Action)
		buf.WriteString(" ")
		buf.WriteString(r.IP)
		buf.WriteString("\n")
	}
	return buf.String()
}

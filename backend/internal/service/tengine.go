package service

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"regexp"

	"tpm/internal/model"
)

type TengineService struct {
	tmplProxy *template.Template
	tmplSSL   *template.Template
	confDir   string
}

func NewTengineService() (*TengineService, error) {
	proxy, err := template.ParseFiles("templates/proxy.conf.tmpl")
	if err != nil {
		return nil, err
	}
	ssl, err := template.ParseFiles("templates/ssl.conf.tmpl")
	if err != nil {
		return nil, err
	}

	confDir := os.Getenv("TENGINE_CONF_DIR")
	if confDir == "" {
		confDir = "/etc/tengine/conf.d"
	}

	return &TengineService{
		tmplProxy: proxy,
		tmplSSL:   ssl,
		confDir:   confDir,
	}, nil
}

// GenerateAndReload writes the config file for a proxy host.
// Tengine container watches conf.d via inotify and auto-reloads.
func (s *TengineService) GenerateAndReload(host model.ProxyHost) error {
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9.-]+$`, host.Domain); !matched {
		return fmt.Errorf("invalid domain for config generation: %s", host.Domain)
	}

	confPath := fmt.Sprintf("%s/%s.conf", s.confDir, host.Domain)

	tmpl := s.tmplProxy
	if host.SslEnabled {
		tmpl = s.tmplSSL
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, host); err != nil {
		return err
	}

	if err := os.WriteFile(confPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

// DeleteConfig removes the config file for a domain.
// Tengine container watches conf.d via inotify and auto-reloads.
func (s *TengineService) DeleteConfig(domain string) error {
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9.-]+$`, domain); !matched {
		return fmt.Errorf("invalid domain for config deletion: %s", domain)
	}

	confPath := fmt.Sprintf("%s/%s.conf", s.confDir, domain)
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

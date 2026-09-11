package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"gorm.io/gorm"
	"tpm/internal/model"
)

// One backend owns config writes. All configuration mutations share this lock.
var configMu sync.Mutex
var configDomain = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`)

type TengineService struct {
	tmplProxy *template.Template
	tmplSSL   *template.Template
	confDir   string
	reload    func() error
}

func ConfigDir() string {
	if dir := os.Getenv("TENGINE_CONF_DIR"); dir != "" {
		return dir
	}
	return "/etc/tengine/conf.d"
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
	s := &TengineService{tmplProxy: proxy, tmplSSL: ssl, confDir: ConfigDir()}
	s.reload = s.requestReload
	return s, nil
}

func (s *TengineService) Render(host model.ProxyHost) ([]byte, error) {
	if !configDomain.MatchString(host.Domain) || strings.Contains(host.Domain, "..") {
		return nil, fmt.Errorf("invalid domain")
	}
	if !host.Enabled {
		return nil, nil
	}
	tmpl := s.tmplProxy
	if host.SslEnabled {
		tmpl = s.tmplSSL
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, host); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// AtomicWrite keeps partial files out of Tengine's include glob.
func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tpm-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

// Transaction stages DB changes, applies files, and waits for Tengine validation
// and reload acknowledgement before committing. Failures restore the previous files.
func (s *TengineService) Transaction(db *gorm.DB, change func(*gorm.DB) (map[string][]byte, error)) error {
	configMu.Lock()
	defer configMu.Unlock()
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()
	changes, err := change(tx)
	if err != nil {
		return err
	}
	return s.apply(changes, func() error { return tx.Commit().Error })
}

func (s *TengineService) Apply(changes map[string][]byte) error {
	configMu.Lock()
	defer configMu.Unlock()
	return s.apply(changes, func() error { return nil })
}

func (s *TengineService) apply(changes map[string][]byte, commit func() error) error {
	old := make(map[string][]byte, len(changes))
	for name := range changes {
		if filepath.Base(name) != name || (!strings.HasSuffix(name, ".conf") && name != "000-default.html") {
			return fmt.Errorf("invalid config filename")
		}
		data, err := os.ReadFile(filepath.Join(s.confDir, name))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		old[name] = data
	}
	write := func(files map[string][]byte) error {
		var errs []error
		for name, data := range files {
			path := filepath.Join(s.confDir, name)
			if data == nil {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					errs = append(errs, err)
				}
			} else if err := AtomicWrite(path, data, 0644); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	rollback := func(cause error) error {
		restoreErr := write(old)
		if restoreErr != nil {
			return errors.Join(cause, fmt.Errorf("config restoration failed: %w", restoreErr))
		}
		if err := s.reload(); err != nil {
			return errors.Join(cause, fmt.Errorf("restored files but reload failed: %w", err))
		}
		return cause
	}
	if err := write(changes); err != nil {
		return rollback(err)
	}
	if len(changes) > 0 {
		if err := s.reload(); err != nil {
			return rollback(err)
		}
	}
	if err := commit(); err != nil {
		return rollback(err)
	}
	return nil
}

// The Tengine container processes requests on the shared volume. No Docker socket
// or externally reachable control API is needed. A deadline rejects stale requests.
func (s *TengineService) requestReload() error {
	dir := filepath.Join(filepath.Dir(s.confDir), "control")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return err
	}
	base := filepath.Join(dir, hex.EncodeToString(id))
	deadline := time.Now().Add(10 * time.Second)
	request, response := base+".request", base+".response"
	defer os.Remove(request)
	defer os.Remove(response)
	if err := AtomicWrite(request, []byte(strconv.FormatInt(deadline.Unix(), 10)), 0600); err != nil {
		return err
	}
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(response)
		if err == nil {
			if strings.TrimSpace(string(data)) == "ok" {
				return nil
			}
			return fmt.Errorf("Tengine rejected configuration: %s", strings.TrimSpace(string(data)))
		}
		if !os.IsNotExist(err) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("Tengine did not acknowledge configuration within 10 seconds")
}

func (s *TengineService) Reload() error {
	configMu.Lock()
	defer configMu.Unlock()
	return s.reload()
}

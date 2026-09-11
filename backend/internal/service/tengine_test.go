package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"tpm/internal/model"
)

func TestConfigRollback(t *testing.T) {
	for _, failure := range []string{"validation", "commit"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			old := filepath.Join(dir, "old.example.com.conf")
			if err := os.WriteFile(old, []byte("previous"), 0644); err != nil {
				t.Fatal(err)
			}
			reloads := 0
			s := &TengineService{confDir: dir, reload: func() error {
				reloads++
				if failure == "validation" && reloads == 1 {
					return errors.New("invalid config")
				}
				return nil
			}}
			committed := false
			err := s.apply(map[string][]byte{"old.example.com.conf": nil, "new.example.com.conf": []byte("candidate")}, func() error { committed = true; return errors.New("DB commit failed") })
			if err == nil {
				t.Fatal("expected rejection")
			}
			data, _ := os.ReadFile(old)
			if string(data) != "previous" {
				t.Fatal("old file not restored")
			}
			if _, err := os.Stat(filepath.Join(dir, "new.example.com.conf")); !os.IsNotExist(err) {
				t.Fatal("candidate was not removed")
			}
			if failure == "validation" && committed {
				t.Fatal("committed a rejected configuration")
			}
			if reloads != 2 {
				t.Fatalf("reloads=%d", reloads)
			}
		})
	}
}

func TestTransactionRollsBackDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "db.sqlite")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	s := &TengineService{confDir: t.TempDir(), reload: func() error { return errors.New("rejected") }}
	err = s.Transaction(db, func(tx *gorm.DB) (map[string][]byte, error) {
		if err := tx.Create(&model.User{Email: "new@example.com", Password: "hash"}).Error; err != nil {
			return nil, err
		}
		return map[string][]byte{"host.conf": []byte("invalid")}, nil
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count != 0 {
		t.Fatal("failed config persisted database changes")
	}
}

func TestRenderAccessRulesAndDisabledHost(t *testing.T) {
	proxy, err := template.ParseFiles("../../templates/proxy.conf.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	ssl, err := template.ParseFiles("../../templates/ssl.conf.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	s := &TengineService{tmplProxy: proxy, tmplSSL: ssl}
	host := model.ProxyHost{Domain: "test.example.com", ForwardHost: "127.0.0.1", ForwardPort: 8080, ForwardScheme: "http", Enabled: true, AccessList: &model.AccessList{Rules: `[{"ip":"10.0.0.0/8","action":"allow"}]`}}
	for _, enabled := range []bool{false, true} {
		host.SslEnabled = enabled
		data, err := s.Render(host)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "allow 10.0.0.0/8;") || !strings.Contains(string(data), "deny all;") {
			t.Fatal("ACL missing")
		}
	}
	host.Enabled = false
	data, err := s.Render(host)
	if err != nil || data != nil {
		t.Fatal("disabled host must not produce config")
	}
}

func TestReloadAcknowledgement(t *testing.T) {
	s := &TengineService{confDir: filepath.Join(t.TempDir(), "conf.d")}
	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			files, _ := filepath.Glob(filepath.Join(filepath.Dir(s.confDir), "control", "*.request"))
			if len(files) > 0 {
				_ = AtomicWrite(strings.TrimSuffix(files[0], ".request")+".response", []byte("ok"), 0600)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	if err := s.requestReload(); err != nil {
		t.Fatal(err)
	}
	<-done
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(s.confDir), "control", "*"))
	if len(files) != 0 {
		t.Fatal("control files were not cleaned up")
	}
}

func TestCloudflareCachedProtection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "db.sqlite")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.CloudflareSettings{}); err != nil {
		t.Fatal(err)
	}
	cfg := model.CloudflareSettings{ID: 1, Enabled: true, IPv4List: "192.0.2.0/24"}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	s := NewCloudflareService(db, &TengineService{confDir: t.TempDir()})
	if err := s.Restore(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(s.tengine.confDir, "cloudflare-geo.conf"))
	if !strings.Contains(string(data), "default 0;") || !strings.Contains(string(data), "192.0.2.0/24 1;") {
		t.Fatal("cached protection not restored")
	}
	cfg.IPv4List = ""
	data, err = renderCloudflareGeo(&cfg)
	if err != nil || !strings.Contains(string(data), "default 0;") {
		t.Fatal("empty enabled list must deny")
	}
	cfg.IPv4List = "bad; directive"
	if _, err := renderCloudflareGeo(&cfg); err == nil {
		t.Fatal("accepted invalid CIDR")
	}
}

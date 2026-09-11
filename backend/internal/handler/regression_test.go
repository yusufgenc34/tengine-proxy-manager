package handler

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	mw "tpm/internal/middleware"
	"tpm/internal/model"
	"tpm/internal/service"
)

func testHandler(t *testing.T) (*Handler, *atomic.Bool) {
	t.Helper()
	t.Chdir("../..")
	dir := t.TempDir()
	t.Setenv("TENGINE_CONF_DIR", filepath.Join(dir, "conf.d"))
	t.Setenv("JWT_SECRET", "regression-only-secret-at-least-32-bytes")
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "db.sqlite")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.ProxyHost{}, &model.Certificate{}, &model.AccessList{}, &model.AuditLog{}, &model.CloudflareSettings{}); err != nil {
		t.Fatal(err)
	}
	h, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.InitializeConfig(); err != nil {
		t.Fatal(err)
	}
	reject := &atomic.Bool{}
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				files, _ := filepath.Glob(filepath.Join(dir, "control", "*.request"))
				for _, file := range files {
					response := strings.TrimSuffix(file, ".request") + ".response"
					if _, err := os.Stat(response); err == nil {
						continue
					}
					result := "ok"
					if reject.Load() {
						result = "invalid configuration"
					}
					_ = service.AtomicWrite(response, []byte(result), 0600)
				}
			}
		}
	}()
	t.Cleanup(func() { close(done); <-stopped })
	return h, reject
}

func call(t *testing.T, fn echo.HandlerFunc, method, body, id string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if id != "" {
		c.SetParamNames("id")
		c.SetParamValues(id)
	}
	if err := fn(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

func TestProxyLifecycleAndAccessList(t *testing.T) {
	h, _ := testHandler(t)
	list := model.AccessList{Name: "private", Rules: `[{"ip":"10.0.0.0/8","action":"allow"}]`}
	if err := h.db.Create(&list).Error; err != nil {
		t.Fatal(err)
	}
	body := `{"domain":"app.example.com","forward_host":"127.0.0.1","forward_port":8080,"access_list_id":` + strconv.Itoa(int(list.ID)) + `}`
	rec := call(t, h.CreateProxyHost, "POST", body, "")
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var host model.ProxyHost
	json.Unmarshal(rec.Body.Bytes(), &host)
	id := strconv.Itoa(int(host.ID))
	path := filepath.Join(service.ConfigDir(), host.Domain+".conf")
	assertACL := func() {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), "allow 10.0.0.0/8;") {
			t.Fatalf("ACL missing: %s %v", data, err)
		}
	}
	assertACL()
	if rec := call(t, h.DisableProxyHost, "POST", "{}", id); rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("disabled config retained")
	}
	if rec := call(t, h.UpdateProxyHost, "PUT", `{"forward_port":8081}`, id); rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("editing reactivated host")
	}
	if rec := call(t, h.EnableProxyHost, "POST", "{}", id); rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	assertACL()
	if rec := call(t, h.UpdateProxyHost, "PUT", `{"domain":"new.example.com"}`, id); rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("old domain config retained")
	}
	path = filepath.Join(service.ConfigDir(), "new.example.com.conf")
	assertACL()
	if rec := call(t, h.UpdateProxyHost, "PUT", `{"access_list_id":null}`, id); rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "deny all;") {
		t.Fatal("explicit null did not detach ACL")
	}
	if rec := call(t, h.DeleteProxyHost, "DELETE", "", id); rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("deleted config retained")
	}
}

func TestRejectedConfigDoesNotPersist(t *testing.T) {
	h, reject := testHandler(t)
	reject.Store(true)
	rec := call(t, h.CreateProxyHost, "POST", `{"domain":"app.example.com","forward_host":"127.0.0.1","forward_port":80}`, "")
	if rec.Code != 500 {
		t.Fatalf("expected failure, got %d", rec.Code)
	}
	var count int64
	h.db.Model(&model.ProxyHost{}).Count(&count)
	if count != 0 {
		t.Fatal("failed create persisted")
	}
	if _, err := os.Stat(filepath.Join(service.ConfigDir(), "app.example.com.conf")); !os.IsNotExist(err) {
		t.Fatal("failed config remained")
	}
}

func TestProxyRejectsInvalidInput(t *testing.T) {
	h, _ := testHandler(t)
	for _, body := range []string{
		`{"domain":"../bad.example.com","forward_host":"localhost","forward_port":80}`,
		`{"domain":"app.example.com","forward_host":"localhost; bad","forward_port":80}`,
		`{"domain":"app.example.com","forward_host":"localhost","forward_port":80,"ssl_enabled":true}`,
		`{"domain":"app.example.com","forward_host":"localhost","forward_port":80,"access_list_id":999}`,
	} {
		rec := call(t, h.CreateProxyHost, "POST", body, "")
		if rec.Code != 400 {
			t.Fatalf("accepted invalid input: %s (%d %s)", body, rec.Code, rec.Body)
		}
	}
	var count int64
	h.db.Model(&model.ProxyHost{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid input persisted")
	}
}

func TestRefreshRejectsAccessAndTempTokens(t *testing.T) {
	h, _ := testHandler(t)
	user := model.User{Email: "admin@example.com", Password: "hash", Role: "admin"}
	h.db.Create(&user)
	for _, kind := range []string{mw.TempToken, mw.AccessToken, mw.RefreshToken} {
		raw, err := generateToken(user, kind, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		rec := call(t, h.Refresh, "POST", `{"refresh_token":"`+raw+`"}`, "")
		want := 401
		if kind == mw.RefreshToken {
			want = 200
		}
		if rec.Code != want {
			t.Fatalf("%s refresh: got %d want %d", kind, rec.Code, want)
		}
	}
}

func TestCustomCertRejectsTraversalAndInvalidPEM(t *testing.T) {
	h, _ := testHandler(t)
	for _, domain := range []string{"../../tmp/escape", "valid.example.com"} {
		rec := call(t, h.UploadCustomCert, "POST", `{"domain":"`+domain+`","cert_content":"invalid","key_content":"invalid"}`, "")
		if rec.Code != 400 {
			t.Fatalf("invalid upload accepted: %d", rec.Code)
		}
	}
	var count int64
	h.db.Model(&model.Certificate{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid certificate persisted")
	}
}

package file

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/spf13/afero"
)

func TestHandlerRejectsTraversalInRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	handler := NewHandler(fs, "/app")
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/file/process?path=/../../etc/passwd&mode=edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Process(c)
	if err == nil {
		t.Fatal("Process() should reject traversal path")
	}
	if !IsPathNotAllowed(err) {
		t.Fatalf("Process() error = %v, want path not allowed", err)
	}
}

func TestHandlerFallsBackToRootWhenCookiePathIsInvalid(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("/app", 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := afero.WriteFile(fs, "/readme.txt", []byte("ok"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	handler := NewHandler(fs, "/app")
	e := echo.New()
	e.Renderer = testRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/admin/file", nil)
	req.AddCookie(&http.Cookie{Name: "dirHistory", Value: "%2F..%2F..%2Fetc"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Index(c); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	found := false
	for _, setCookie := range rec.Header().Values("Set-Cookie") {
		if strings.Contains(setCookie, "dirHistory=%2F") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Set-Cookie headers = %q, want dirHistory to be reset to %%2F", rec.Header().Values("Set-Cookie"))
	}
}

func TestHandlerSupportsChmodMode(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("/app", 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := afero.WriteFile(fs, "/app/note.txt", []byte("ok"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	handler := NewHandler(fs, "/app")
	e := echo.New()

	req := httptest.NewRequest(http.MethodPut, "/admin/file/process?path=/note.txt&mode=chmod", strings.NewReader("0600"))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Process(c); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(rec.Body.String(), `"message":"success"`) {
		t.Fatalf("Process() body = %q, want success message", rec.Body.String())
	}
	info, err := fs.Stat("/app/note.txt")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode = %v, want 0600", got)
	}
}

type testRenderer struct{}

func (testRenderer) Render(_ *echo.Context, w io.Writer, _ string, _ any) error {
	_, err := io.WriteString(w, "ok")
	return err
}

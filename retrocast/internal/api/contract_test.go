package api

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"go.yaml.in/yaml/v3"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every routed handler must use the common success writer or an empty 204.
// Existing handler tests exercise these writers with their service fixtures.
func TestContractAllHandlers(t *testing.T) {
	files, err := filepath.Glob("*_handler.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(source), "c.JSON(") {
				t.Fatal("handler bypasses REST envelope")
			}
		})
	}
}
func TestContractErrors(t *testing.T) {
	e := echo.New()
	tokens := auth.NewTokenService("test-secret")
	SetupRouter(e, &Dependencies{TokenService: tokens})
	e.GET("/protected", func(c echo.Context) error { return c.NoContent(204) }, tokens.Middleware())
	e.POST("/large", func(c echo.Context) error { var v any; return c.Bind(&v) }, middleware.BodyLimit("1B"))
	e.GET("/internal", func(c echo.Context) error { return echo.NewHTTPError(500, "private database detail") })
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/protected", "", 401}, {"GET", "/unknown", "", 404}, {"POST", "/protected", "", 405}, {"POST", "/large", "{}", 413}, {"GET", "/internal", "", 500},
	} {
		t.Run(tc.path+tc.method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d", rec.Code)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body["error"] == nil {
				t.Fatal("missing error envelope")
			}
			var detail map[string]string
			if err := json.Unmarshal(body["error"], &detail); err != nil {
				t.Fatal(err)
			}
			if len(detail) != 2 || detail["code"] == "" || detail["message"] == "" {
				t.Fatal("invalid error detail")
			}
			if strings.Contains(rec.Body.String(), "private") {
				t.Fatal("leaked internal detail")
			}
		})
	}
}
func TestContractIDs(t *testing.T) {
	const id int64 = 9007199254740993
	for _, value := range []any{models.Member{Roles: []int64{id}}, gateway.ReadyData{Guilds: []int64{id}}} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `["9007199254740993"]`) {
			t.Fatalf("numeric id array: %s", data)
		}
	}
}
func TestContractEmptySuccess(t *testing.T) {
	e := echo.New()
	e.DELETE("/empty", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("DELETE", "/empty", nil))
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatal("204 has a body")
	}
}

func TestContractOpenAPI(t *testing.T) {
	var spec map[string]any
	data, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if spec["info"].(map[string]any)["version"] != "2.0.0" {
		t.Fatal("wrong version")
	}
	count := 0
	for path, raw := range spec["paths"].(map[string]any) {
		for method, rawOp := range raw.(map[string]any) {
			op, ok := rawOp.(map[string]any)
			if !ok {
				continue
			}
			responses, ok := op["responses"].(map[string]any)
			if !ok {
				continue
			}
			for status, rawResponse := range responses {
				if !strings.HasPrefix(status, "2") {
					continue
				}
				count++
				response := rawResponse.(map[string]any)
				content, _ := response["content"].(map[string]any)
				if status == "204" && len(content) > 0 {
					t.Errorf("%s %s: 204 content", method, path)
				}
				for _, rawContent := range content {
					schema := rawContent.(map[string]any)["schema"].(map[string]any)
					properties, _ := schema["properties"].(map[string]any)
					if schema["type"] != "object" || len(properties) != 1 || properties["data"] == nil {
						t.Errorf("%s %s: missing data envelope", method, path)
					}
				}
			}
		}
	}
	if count != 58 {
		t.Fatalf("expected 58 operations, got %d", count)
	}
}

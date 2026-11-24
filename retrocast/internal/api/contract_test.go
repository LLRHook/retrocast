package api

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
	"go.yaml.in/yaml/v3"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
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
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldRegistry })
	e := echo.New()
	tokens := auth.NewTokenService("test-secret")
	SetupRouter(e, &Dependencies{TokenService: tokens})
	e.GET("/protected", func(c echo.Context) error { return c.NoContent(204) }, tokens.Middleware())
	e.POST("/large", func(c echo.Context) error { var v any; return c.Bind(&v) }, middleware.BodyLimit("1B"))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{DisablePrintStack: true}))
	e.GET("/panic", func(c echo.Context) error { panic("private fixture panic") })
	e.GET("/internal", func(c echo.Context) error { return echo.NewHTTPError(500, "private database detail") })
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/protected", "", 401}, {"GET", "/unknown", "", 404}, {"POST", "/protected", "", 405}, {"POST", "/large", "{}", 413}, {"GET", "/internal", "", 500}, {"GET", "/panic", "", 500},
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
	contract, err := os.ReadFile("../../../docs/DATA_CONTRACTS.md")
	if err != nil {
		t.Fatal(err)
	}
	version := regexp.MustCompile("(?m)^Current REST contract version: `([0-9]+\\.[0-9]+\\.[0-9]+)`\\.\\r?$").FindAllSubmatch(contract, -1)
	if len(version) != 1 {
		t.Fatal("missing single current contract version")
	}
	if spec["info"].(map[string]any)["version"] != string(version[0][1]) {
		t.Fatal("OpenAPI version differs from DATA_CONTRACTS current version")
	}
	license := spec["info"].(map[string]any)["license"].(map[string]any)
	if len(license) != 1 || license["name"] != "Proprietary. All rights reserved." {
		t.Fatal("license mismatch", license)
	}
	// Shared response/schema references must resolve, including API-059 errors.
	var checkRefs func(any)
	checkRefs = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
				var target any = spec
				for _, key := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
					m, ok := target.(map[string]any)
					if !ok {
						t.Errorf("unresolved OpenAPI ref %s", ref)
						break
					}
					target, ok = m[key]
					if !ok {
						t.Errorf("unresolved OpenAPI ref %s", ref)
						break
					}
				}
			}
			for _, child := range v {
				checkRefs(child)
			}
		case []any:
			for _, child := range v {
				checkRefs(child)
			}
		}
	}
	checkRefs(spec)
	checkIDSchemas(t, spec, "OpenAPI", false)
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
			if id, ok := op["operationId"].(string); !ok || id == "" {
				t.Errorf("%s %s: missing operationId", method, path)
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
					if op["operationId"] == "changePassword" {
						required, ok := schema["required"].([]any)
						if !ok || len(required) != 1 || required[0] != "data" || schema["additionalProperties"] != false {
							t.Error("password response must require only the data envelope")
						}
					}
					if op["operationId"] == "getReactions" {
						checkIDSchemas(t, properties["data"], "getReactions/data", true)
					}
					if schema["type"] != "object" || len(properties) != 1 || properties["data"] == nil {
						t.Errorf("%s %s: missing data envelope", method, path)
					}
				}
			}
		}
	}
	if count != 60 {
		t.Fatalf("expected 60 operations, got %d", count)
	}
}

func TestHTTPErrorCodes(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{
		{400, "BAD_REQUEST"}, {401, "UNAUTHORIZED"}, {403, "FORBIDDEN"}, {404, "NOT_FOUND"}, {405, "METHOD_NOT_ALLOWED"}, {413, "PAYLOAD_TOO_LARGE"}, {429, "RATE_LIMITED"},
		{402, "BAD_REQUEST"}, {409, "BAD_REQUEST"}, {410, "BAD_REQUEST"}, {415, "BAD_REQUEST"}, {416, "BAD_REQUEST"}, {422, "BAD_REQUEST"}, {499, "BAD_REQUEST"}, {500, "INTERNAL"}, {503, "INTERNAL"},
	} {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest("GET", "/error", nil), rec)
			HTTPErrorHandler(echo.NewHTTPError(tc.status, "fixture error"), c)
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.status || body.Error.Code != tc.code {
				t.Fatalf("status/code = %d/%s, want %d/%s", rec.Code, body.Error.Code, tc.status, tc.code)
			}
		})
	}
}

// Components are walked as well as inline schemas, so referenced id definitions
// are checked once at their source. Id array items inherit identifier context.
func checkIDSchemas(t *testing.T, value any, path string, identifier bool) {
	t.Helper()
	switch node := value.(type) {
	case map[string]any:
		if identifier && node["type"] == "integer" {
			t.Errorf("%s declares a numeric snowflake", path)
		}
		for key, child := range node {
			if key == "properties" {
				for name, schema := range child.(map[string]any) {
					isID := name == "id" || strings.HasSuffix(name, "_id") || strings.HasSuffix(name, "_ids") || name == "roles"
					checkIDSchemas(t, schema, path+"/"+name, isID)
				}
			} else {
				checkIDSchemas(t, child, path+"/"+key, identifier)
			}
		}
	case []any:
		for _, child := range node {
			checkIDSchemas(t, child, path, identifier)
		}
	}
}

func TestContractRouteInventory(t *testing.T) {
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldRegistry })
	e := echo.New()
	SetupRouter(e, &Dependencies{TokenService: auth.NewTokenService("test-secret")})
	// Independent, explicit cases prevent a newly registered route from silently
	// bypassing contract coverage. Parameter names can differ in router and spec.
	cases := map[string]int{
		"PUT /api/v1/guilds/:param/roles/order":                               200,
		"PUT /api/v1/users/@me/password":                                      200,
		"POST /api/v1/auth/register":                                          200,
		"POST /api/v1/auth/login":                                             200,
		"POST /api/v1/auth/refresh":                                           200,
		"GET /api/v1/invites/:param":                                          200,
		"POST /api/v1/auth/logout":                                            204,
		"GET /api/v1/users/@me":                                               200,
		"PATCH /api/v1/users/@me":                                             200,
		"GET /api/v1/users/@me/guilds":                                        200,
		"POST /api/v1/users/@me/channels":                                     200,
		"GET /api/v1/users/@me/channels":                                      200,
		"PUT /api/v1/channels/:param/recipients/:param":                       204,
		"DELETE /api/v1/channels/:param/recipients/:param":                    204,
		"GET /api/v1/users/@me/read-states":                                   200,
		"POST /api/v1/guilds":                                                 201,
		"GET /api/v1/guilds/:param":                                           200,
		"PATCH /api/v1/guilds/:param":                                         200,
		"DELETE /api/v1/guilds/:param":                                        204,
		"POST /api/v1/guilds/:param/channels":                                 201,
		"GET /api/v1/guilds/:param/channels":                                  200,
		"GET /api/v1/channels/:param":                                         200,
		"PATCH /api/v1/channels/:param":                                       200,
		"DELETE /api/v1/channels/:param":                                      204,
		"GET /api/v1/guilds/:param/members":                                   200,
		"GET /api/v1/guilds/:param/members/:param":                            200,
		"PATCH /api/v1/guilds/:param/members/:param":                          200,
		"PATCH /api/v1/guilds/:param/members/@me":                             200,
		"DELETE /api/v1/guilds/:param/members/:param":                         204,
		"DELETE /api/v1/guilds/:param/members/@me":                            204,
		"POST /api/v1/guilds/:param/roles":                                    201,
		"GET /api/v1/guilds/:param/roles":                                     200,
		"PATCH /api/v1/guilds/:param/roles/:param":                            200,
		"DELETE /api/v1/guilds/:param/roles/:param":                           204,
		"PUT /api/v1/guilds/:param/members/:param/roles/:param":               204,
		"DELETE /api/v1/guilds/:param/members/:param/roles/:param":            204,
		"PUT /api/v1/channels/:param/permissions/:param":                      200,
		"DELETE /api/v1/channels/:param/permissions/:param":                   204,
		"POST /api/v1/channels/:param/messages":                               201,
		"GET /api/v1/channels/:param/messages":                                200,
		"GET /api/v1/channels/:param/messages/:param":                         200,
		"PATCH /api/v1/channels/:param/messages/:param":                       200,
		"DELETE /api/v1/channels/:param/messages/:param":                      204,
		"GET /api/v1/guilds/:param/messages/search":                           200,
		"PUT /api/v1/channels/:param/messages/:param/reactions/:param/@me":    204,
		"DELETE /api/v1/channels/:param/messages/:param/reactions/:param/@me": 204,
		"GET /api/v1/channels/:param/messages/:param/reactions/:param":        200,
		"POST /api/v1/channels/:param/voice/join":                             200,
		"POST /api/v1/channels/:param/voice/leave":                            204,
		"GET /api/v1/channels/:param/voice/states":                            200,
		"POST /api/v1/channels/:param/attachments":                            201,
		"POST /api/v1/channels/:param/typing":                                 204,
		"PUT /api/v1/channels/:param/ack/:param":                              204,
		"PUT /api/v1/guilds/:param/bans/:param":                               204,
		"DELETE /api/v1/guilds/:param/bans/:param":                            204,
		"GET /api/v1/guilds/:param/bans":                                      200,
		"POST /api/v1/guilds/:param/invites":                                  201,
		"GET /api/v1/guilds/:param/invites":                                   200,
		"POST /api/v1/invites/:param":                                         200,
		"DELETE /api/v1/invites/:param":                                       204,
	}
	exclusions := map[string]string{
		echo.RouteNotFound + " /api/v1":        "Echo protected-group fallback",
		echo.RouteNotFound + " /api/v1/*":      "Echo protected-group wildcard fallback",
		echo.RouteNotFound + " /api/v1/auth":   "Echo auth-group fallback",
		echo.RouteNotFound + " /api/v1/auth/*": "Echo auth-group wildcard fallback",
	}
	var spec map[string]any
	data, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	parameter := regexp.MustCompile(`:[^/]+|\{[^}]+\}`)
	normalize := func(path string) string { return parameter.ReplaceAllString(path, ":param") }
	schemas := map[string]map[string]any{}
	for path, raw := range spec["paths"].(map[string]any) {
		for method, rawOp := range raw.(map[string]any) {
			if op, ok := rawOp.(map[string]any); ok && op["responses"] != nil {
				schemas[strings.ToUpper(method)+" "+normalize("/api/v1"+path)] = op
			}
		}
	}
	seen := map[string]bool{}
	for _, route := range e.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1") {
			continue
		}
		rawKey := route.Method + " " + route.Path
		if _, ok := exclusions[rawKey]; ok {
			continue
		}
		key := route.Method + " " + normalize(route.Path)
		expected, ok := cases[key]
		if !ok {
			t.Errorf("registered route has no contract case: %s", rawKey)
			continue
		}
		seen[key] = true
		t.Run(key, func(t *testing.T) {
			op, ok := schemas[key]
			if !ok {
				t.Fatalf("contract case has no schema: %s", key)
			}
			rawResponse, ok := op["responses"].(map[string]any)[strconv.Itoa(expected)]
			if !ok {
				t.Fatalf("missing success status %d", expected)
			}
			content, _ := rawResponse.(map[string]any)["content"].(map[string]any)
			if expected == 204 {
				if len(content) != 0 {
					t.Fatal("204 declares a body")
				}
				return
			}
			schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
			props := schema["properties"].(map[string]any)
			if len(props) != 1 || props["data"] == nil {
				t.Fatal("success schema must contain only data")
			}
		})
	}
	for key := range cases {
		if !seen[key] {
			t.Errorf("contract case is not registered: %s", key)
		}
	}
}

// Compare the actual error mapper to the documented session response, rather
// than duplicating its status number in the assertion.
func TestContractSessionUnavailable(t *testing.T) {
	var spec map[string]any
	data, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err = yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	c, rec := newTestContext(http.MethodPost, "/", nil)
	if err := mapServiceError(c, service.Unavailable()); err != nil {
		t.Fatal(err)
	}
	var body ErrorResponse
	if err := decodeResponse(rec, &body); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ path, method string }{{"/auth/register", "post"}, {"/auth/login", "post"}, {"/auth/refresh", "post"}, {"/auth/logout", "post"}, {"/users/@me/password", "put"}} {
		t.Run(route.path, func(t *testing.T) {
			op := spec["paths"].(map[string]any)[route.path].(map[string]any)[route.method].(map[string]any)
			response, ok := op["responses"].(map[string]any)[strconv.Itoa(rec.Code)].(map[string]any)
			if !ok {
				t.Fatal("mapped session status missing from OpenAPI")
			}
			if !strings.Contains(response["description"].(string), body.Error.Code) {
				t.Fatal("mapped session code missing")
			}
			schema := response["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			if schema["$ref"] != "#/components/schemas/ErrorResponse" {
				t.Fatal("session error does not use shared schema")
			}
		})
	}
}

// TestContractDocumentedSuccess enumerates the document, so a new operation
// cannot silently escape runtime status, envelope and response schema checks.
func TestContractDocumentedSuccess(t *testing.T) {
	var spec map[string]any
	data, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err = yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	bodies := map[string]string{
		"register": `{"username":"newuser","password":"password123"}`, "login": `{"username":"owner","password":"password123"}`,
		"refreshToken": `{"refresh_token":"pilot-refresh"}`, "logout": `{"refresh_token":"pilot-refresh"}`,
		"changePassword": `{"current_password":"password123","new_password":"changed123"}`,
		"updateMe":       `{"display_name":"Changed"}`, "createDM": `{"recipient_id":"200"}`,
		"createGuild": `{"name":"New guild"}`, "updateGuild": `{"name":"Renamed"}`,
		"createChannel": `{"name":"random","type":0}`, "updateChannel": `{"name":"renamed"}`,
		"sendMessage": `{"content":"hi"}`, "editMessage": `{"content":"changed"}`,
		"updateSelf": `{"nickname":"Buddy"}`, "updateMember": `{"nickname":"Buddy"}`,
		"createRole": `{"name":"New role"}`, "updateRole": `{"name":"Changed"}`,
		"reorderRoles":       `{"expected":[{"id":"30","position":0},{"id":"31","position":1}],"order":["31"]}`,
		"setChannelOverride": `{"allow":"0","deny":"0"}`, "createInvite": `{}`, "banMember": `{"reason":"spam"}`,
	}
	count := 0
	for path, raw := range spec["paths"].(map[string]any) {
		for method, rawOp := range raw.(map[string]any) {
			op, ok := rawOp.(map[string]any)
			if !ok || op["responses"] == nil {
				continue
			}
			count++
			id := op["operationId"].(string)
			t.Run(id, func(t *testing.T) {
				e, token, _ := pilotContractServer(t)
				target := path
				params := map[string]string{"guildId": "1", "channelId": "10", "messageId": "20", "roleId": "31", "userId": "200", "code": "pilot", "emoji": "smile"}
				for key, value := range params {
					target = strings.ReplaceAll(target, "{"+key+"}", value)
				}
				switch id {
				case "leaveGuild":
					target = strings.ReplaceAll(target, "/guilds/1/", "/guilds/2/")
				case "addGroupDMMember", "removeGroupDMMember":
					target = strings.ReplaceAll(target, "/channels/10/", "/channels/12/")
					if id == "addGroupDMMember" {
						target = strings.ReplaceAll(target, "/recipients/200", "/recipients/300")
					}
				case "joinVoice", "leaveVoice", "getVoiceStates":
					target = strings.ReplaceAll(target, "/channels/10/", "/channels/11/")
				case "searchMessages":
					target += "?q=hello"
				}
				req := httptest.NewRequest(strings.ToUpper(method), "/api/v1"+target, strings.NewReader(bodies[id]))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)
				if id == "acceptInvite" {
					token, _ = auth.NewTokenService("pilot-contract-fixture").GenerateAccessToken(300)
					req.Header.Set("Authorization", "Bearer "+token)
				}
				if id == "uploadAttachment" {
					c, _ := newMultipartContext(t, "photo.png", "image/png", []byte("fixture png"))
					req = c.Request()
					req.URL.Path = "/api/v1" + target
					req.Method = strings.ToUpper(method)
					req.Header.Set("Authorization", "Bearer "+token)
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				responses := op["responses"].(map[string]any)
				var status string
				for code := range responses {
					if strings.HasPrefix(code, "2") {
						if status != "" {
							t.Fatal("multiple successes require explicit fixture")
						}
						status = code
					}
				}
				if strconv.Itoa(rec.Code) != status {
					t.Fatalf("%s %s: status %d want %s: %s", method, target, rec.Code, status, rec.Body.String())
				}
				if status == "204" {
					if rec.Body.Len() != 0 {
						t.Fatal("204 response has body")
					}
					return
				}
				var body any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				envelope, ok := body.(map[string]any)
				if !ok || len(envelope) != 1 || envelope["data"] == nil {
					t.Fatalf("data envelope: %s", rec.Body.String())
				}
				schema := responses[status].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
				validateContractSchema(t, spec, schema, body, "response")
			})
		}
	}
	if count != 60 {
		t.Fatalf("covered %d operations, want 60; skipped 0", count)
	}
}

func validateContractSchema(t *testing.T, spec, schema map[string]any, value any, path string) {
	t.Helper()
	if all, ok := schema["allOf"].([]any); ok {
		for _, child := range all {
			validateContractSchema(t, spec, child.(map[string]any), value, path)
		}
	}
	if ref, ok := schema["$ref"].(string); ok {
		var node any = spec
		for _, key := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
			node = node.(map[string]any)[key]
		}
		validateContractSchema(t, spec, node.(map[string]any), value, path)
		return
	}
	if value == nil {
		if schema["nullable"] != true {
			t.Errorf("%s: unexpected null", path)
		}
		return
	}
	switch schema["type"] {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			t.Errorf("%s: want object got %T", path, value)
			return
		}
		props, _ := schema["properties"].(map[string]any)
		if required, ok := schema["required"].([]any); ok {
			for _, key := range required {
				if _, ok := obj[key.(string)]; !ok {
					t.Errorf("%s: missing %s", path, key)
				}
			}
		}
		for key, v := range obj {
			if child, ok := props[key].(map[string]any); ok {
				validateContractSchema(t, spec, child, v, path+"/"+key)
			} else if schema["additionalProperties"] == false {
				t.Errorf("%s: extra %s", path, key)
			}
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			t.Errorf("%s: want array got %T", path, value)
			return
		}
		for i, v := range arr {
			validateContractSchema(t, spec, schema["items"].(map[string]any), v, fmt.Sprintf("%s/%d", path, i))
		}
	case "string":
		v, ok := value.(string)
		if !ok {
			t.Errorf("%s: want string got %T", path, value)
			return
		}
		if pattern, ok := schema["pattern"].(string); ok && !regexp.MustCompile(pattern).MatchString(v) {
			t.Errorf("%s: pattern mismatch", path)
		}
		if schema["format"] == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, v); err != nil {
				t.Errorf("%s: invalid date-time", path)
			}
		}
	case "integer":
		n, ok := value.(float64)
		if !ok || n != math.Trunc(n) {
			t.Errorf("%s: want integer", path)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			t.Errorf("%s: want number", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			t.Errorf("%s: want boolean", path)
		}
	}
	if enum, ok := schema["enum"].([]any); ok {
		found := false
		for _, v := range enum {
			if fmt.Sprint(v) == fmt.Sprint(value) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: value outside enum", path)
		}
	}
}

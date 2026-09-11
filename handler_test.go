package version

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	handler := Handler(HandlerConfig{Info: info, IncludeBuildDetails: true})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var parsed Info
	err = json.Unmarshal(body, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
	// Build detail is opt-in now; see TestHandlerOmitsBuildDetailsByDefault.
	assert.Equal(t, "abc123", parsed.Commit)
}

func TestHandler_DefaultConfig(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var parsed Info
	err = json.Unmarshal(body, &parsed)
	require.NoError(t, err)

	// Should use default version
	assert.NotEmpty(t, parsed.Version)
}

func TestHandler_WithHeaders(t *testing.T) {
	info := NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")
	handler := Handler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-App-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
	assert.Equal(t, "2025-01-01T00:00:00Z", resp.Header.Get("X-App-Build-Date"))
}

func TestHandler_WithHeaders_SanitizesValues(t *testing.T) {
	info := NewWithBranch("1.0.0\r\n", "abc1234567890", "2025-01-01T00:00:00Z\r\n", "main\r\n")
	handler := Handler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
	assert.Equal(t, "2025-01-01T00:00:00Z", resp.Header.Get("X-App-Build-Date"))
}

func TestHandler_Pretty(t *testing.T) {
	info := New("1.0.0", "abc123", "")
	handler := Handler(HandlerConfig{
		Info:                info,
		Pretty:              true,
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Pretty output should contain indentation
	assert.Contains(t, string(body), "\n")
	assert.Contains(t, string(body), "  ")
}

func TestTextHandler(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	handler := TextHandler(HandlerConfig{Info: info, IncludeBuildDetails: true})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Version:    1.0.0")
	assert.Contains(t, string(body), "Commit:     abc123")
}

func TestSimpleHandler(t *testing.T) {
	// Save and restore original values
	origVersion := Version
	origCommit := Commit
	defer func() {
		Version = origVersion
		Commit = origCommit
	}()

	Version = "2.0.0"
	Commit = "xyz789"

	handler := SimpleHandler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "2.0.0")
}

func TestMiddleware(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	middleware := Middleware(info, "X-")

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := middleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
	assert.Equal(t, "abc123", resp.Header.Get("X-Commit"))
}

func TestMiddleware_DefaultInfo(t *testing.T) {
	middleware := Middleware(nil, "")

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should have default X-Version header
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestRegisterEndpoint(t *testing.T) {
	info := New("1.0.0", "abc123", "")
	mux := http.NewServeMux()
	RegisterEndpoint(mux, "/version", HandlerConfig{Info: info, IncludeBuildDetails: true})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/version")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestDefaultHandlerConfig(t *testing.T) {
	cfg := DefaultHandlerConfig()

	assert.NotNil(t, cfg.Info)
	assert.False(t, cfg.Pretty)
	assert.False(t, cfg.IncludeHeaders)
	assert.Equal(t, "X-", cfg.HeaderPrefix)
}

// Fiber tests

func TestFiberHandler(t *testing.T) {
	app := fiber.New()
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	app.Get("/version", FiberHandler(HandlerConfig{Info: info, IncludeBuildDetails: true}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
	// Build detail is opt-in now; see TestHandlerOmitsBuildDetailsByDefault.
	assert.Equal(t, "abc123", parsed.Commit)
}

func TestFiberHandler_WithHeaders(t *testing.T) {
	app := fiber.New()
	info := NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")

	app.Get("/version", FiberHandler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-",
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-Branch"))
}

func TestFiberTextHandler(t *testing.T) {
	app := fiber.New()
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	app.Get("/version", FiberTextHandler(HandlerConfig{Info: info, IncludeBuildDetails: true}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Version:    1.0.0")
}

func TestFiberSimpleHandler(t *testing.T) {
	// Save and restore original values
	origVersion := Version
	origCommit := Commit
	defer func() {
		Version = origVersion
		Commit = origCommit
	}()

	Version = "2.0.0"
	Commit = "xyz789"

	app := fiber.New()
	app.Get("/version", FiberSimpleHandler())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "2.0.0")
}

func TestFiberMiddleware(t *testing.T) {
	app := fiber.New()
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	app.Use(FiberMiddleware(info, "X-"))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
	assert.Equal(t, "abc123", resp.Header.Get("X-Commit"))
}

func TestFiberMiddleware_DefaultInfo(t *testing.T) {
	app := fiber.New()

	app.Use(FiberMiddleware(nil, ""))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should have default X-Version header
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestRegisterEndpointFiber(t *testing.T) {
	app := fiber.New()
	info := New("1.0.0", "abc123", "")

	RegisterEndpointFiber(app, "/version", HandlerConfig{Info: info, IncludeBuildDetails: true})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestHandler_NoCommit(t *testing.T) {
	info := &Info{
		Version: "1.0.0",
	}
	handler := Handler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should not have commit header when commit is empty
	assert.Empty(t, resp.Header.Get("X-Commit"))
}

func TestHandler_NilInfoAndEmptyPrefix(t *testing.T) {
	// Test with nil Info (should use Default) and empty HeaderPrefix (should use "X-")
	handler := Handler(HandlerConfig{
		Info:                nil,
		IncludeHeaders:      true,
		HeaderPrefix:        "", // Empty prefix should default to "X-"
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Should have X- prefix headers
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestFiberHandler_NilInfoAndEmptyPrefix(t *testing.T) {
	app := fiber.New()

	// Test with nil Info (should use Default) and empty HeaderPrefix (should use "X-")
	app.Get("/version", FiberHandler(HandlerConfig{
		Info:                nil,
		IncludeHeaders:      true,
		HeaderPrefix:        "", // Empty prefix should default to "X-"
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Should have X- prefix headers
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestFiberHandler_DefaultConfig(t *testing.T) {
	app := fiber.New()
	app.Get("/version", FiberHandler())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	// Should use default version
	assert.NotEmpty(t, parsed.Version)
}

func TestFiberHandler_Pretty(t *testing.T) {
	app := fiber.New()
	info := New("1.0.0", "abc123", "")

	app.Get("/version", FiberHandler(HandlerConfig{
		Info:                info,
		Pretty:              true,
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestTextHandler_DefaultConfig(t *testing.T) {
	handler := TextHandler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info
	assert.Contains(t, string(body), "Version:")
}

func TestTextHandler_WithHeaders(t *testing.T) {
	info := NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")
	handler := TextHandler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-App-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
}

func TestTextHandler_NilInfo(t *testing.T) {
	handler := TextHandler(HandlerConfig{
		Info:                nil, // Should use Default()
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info from Default()
	assert.Contains(t, string(body), "Version:")
}

func TestFiberTextHandler_DefaultConfig(t *testing.T) {
	app := fiber.New()
	app.Get("/version", FiberTextHandler())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info
	assert.Contains(t, string(body), "Version:")
}

func TestFiberTextHandler_WithHeaders(t *testing.T) {
	app := fiber.New()
	info := NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")

	app.Get("/version", FiberTextHandler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-App-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
}

func TestFiberTextHandler_NilInfo(t *testing.T) {
	app := fiber.New()
	app.Get("/version", FiberTextHandler(HandlerConfig{
		Info:                nil, // Should use Default()
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info from Default()
	assert.Contains(t, string(body), "Version:")
}

func TestSetVersionHeaders_UnknownCommit(t *testing.T) {
	info := &Info{
		Version:   "1.0.0",
		Commit:    "unknown",
		BuildDate: "unknown",
	}
	handler := Handler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should not have commit/build-date headers when they are "unknown"
	assert.Empty(t, resp.Header.Get("X-Commit"))
	assert.Empty(t, resp.Header.Get("X-Build-Date"))
}

func TestSetVersionHeadersFiber_UnknownCommit(t *testing.T) {
	app := fiber.New()
	info := &Info{
		Version:   "1.0.0",
		Commit:    "unknown",
		BuildDate: "unknown",
	}

	app.Get("/version", FiberHandler(HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should not have commit/build-date headers when they are "unknown"
	assert.Empty(t, resp.Header.Get("X-Commit"))
	assert.Empty(t, resp.Header.Get("X-Build-Date"))
}

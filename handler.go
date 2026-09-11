package version

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// HandlerConfig configures the version endpoint handler.
type HandlerConfig struct {
	// Info is the version information to return.
	// If nil, Default() will be used.
	Info *Info

	// Pretty enables pretty-printed JSON output.
	// Default: false
	Pretty bool

	// IncludeHeaders adds version info to response headers.
	// Default: false
	IncludeHeaders bool

	// HeaderPrefix is the prefix for version headers.
	// Default: "X-"
	//
	// An invalid prefix (one containing characters that cannot appear in an
	// HTTP header name) falls back to "X-" rather than emitting a malformed
	// header name.
	HeaderPrefix string

	// IncludeBuildDetails serves the full Info -- Go runtime version, commit,
	// build date, platform and compiler.
	//
	// Default: false. This endpoint is usually unauthenticated, and
	// go_version lets anyone match a published Go runtime CVE to the exact
	// build serving them. Turn it on for an internal endpoint, or behind
	// authentication.
	IncludeBuildDetails bool
}

// validHeaderPrefix reports whether prefix can appear in an HTTP header name.
//
// A field name is a token (RFC 9110 5.6.2), so every tchar is allowed here --
// not just the alphanumeric/dash subset. Prefixes such as "X.App-" were valid
// before this validator existed and must keep working.
func validHeaderPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	for _, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
		default:
			return false
		}
	}
	return true
}

// normalizeHeaderPrefix falls back to "X-" for a prefix that is not a token.
func normalizeHeaderPrefix(prefix string) string {
	if validHeaderPrefix(prefix) {
		return prefix
	}
	return "X-"
}

// payload returns the Info to serve, reduced unless build details were asked for.
func (c HandlerConfig) payload() *Info {
	if c.IncludeBuildDetails {
		return c.Info
	}
	return c.Info.Public()
}

// textPayload is payload rendered for the plain-text handlers.
func (c HandlerConfig) textPayload() string {
	if c.IncludeBuildDetails {
		return c.Info.Full()
	}
	return c.Info.Public().Full()
}

// DefaultHandlerConfig returns a HandlerConfig with default values.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		Info:           Default(),
		Pretty:         false,
		IncludeHeaders: false,
		HeaderPrefix:   "X-",
	}
}

// Handler returns an http.HandlerFunc that serves version information.
func Handler(config ...HandlerConfig) http.HandlerFunc {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	cfg.HeaderPrefix = normalizeHeaderPrefix(cfg.HeaderPrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if cfg.IncludeHeaders {
			setVersionHeaders(w.Header(), cfg.payload(), cfg.HeaderPrefix)
		}

		var output []byte
		var err error

		if cfg.Pretty {
			output, err = json.MarshalIndent(cfg.payload(), "", "  ")
		} else {
			output, err = json.Marshal(cfg.payload())
		}

		if err != nil {
			http.Error(w, `{"error": "failed to marshal version info"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}

// FiberHandler returns a Fiber handler that serves version information.
func FiberHandler(config ...HandlerConfig) fiber.Handler {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	cfg.HeaderPrefix = normalizeHeaderPrefix(cfg.HeaderPrefix)

	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/json")

		if cfg.IncludeHeaders {
			setVersionHeadersFiber(c, cfg.payload(), cfg.HeaderPrefix)
		}

		if cfg.Pretty {
			return c.JSON(cfg.payload())
		}

		return c.JSON(cfg.payload())
	}
}

// RegisterEndpoint registers the version handler on an http.ServeMux.
func RegisterEndpoint(mux *http.ServeMux, path string, config ...HandlerConfig) {
	mux.HandleFunc(path, Handler(config...))
}

// RegisterEndpointFiber registers the version handler on a Fiber app.
func RegisterEndpointFiber(app *fiber.App, path string, config ...HandlerConfig) {
	app.Get(path, FiberHandler(config...))
}

// setVersionHeaders adds version information to HTTP headers.
func setVersionHeaders(h http.Header, info *Info, prefix string) {
	h.Set(prefix+"Version", sanitizeHeaderValue(info.Version))

	if info.Commit != "" && info.Commit != "unknown" {
		h.Set(prefix+"Commit", sanitizeHeaderValue(info.ShortCommit()))
	}

	if info.Branch != "" {
		h.Set(prefix+"Branch", sanitizeHeaderValue(info.Branch))
	}

	if info.BuildDate != "" && info.BuildDate != "unknown" {
		h.Set(prefix+"Build-Date", sanitizeHeaderValue(info.BuildDate))
	}
}

// setVersionHeadersFiber adds version information to Fiber response headers.
func setVersionHeadersFiber(c fiber.Ctx, info *Info, prefix string) {
	c.Set(prefix+"Version", sanitizeHeaderValue(info.Version))

	if info.Commit != "" && info.Commit != "unknown" {
		c.Set(prefix+"Commit", sanitizeHeaderValue(info.ShortCommit()))
	}

	if info.Branch != "" {
		c.Set(prefix+"Branch", sanitizeHeaderValue(info.Branch))
	}

	if info.BuildDate != "" && info.BuildDate != "unknown" {
		c.Set(prefix+"Build-Date", sanitizeHeaderValue(info.BuildDate))
	}
}

func sanitizeHeaderValue(value string) string {
	return strings.Map(func(r rune) rune {
		if r <= 31 || r == 127 {
			return -1
		}
		return r
	}, value)
}

// Middleware returns an http.Handler middleware that adds version headers to all responses.
func Middleware(info *Info, prefix string) func(http.Handler) http.Handler {
	if info == nil {
		info = Default()
	}
	if prefix == "" {
		prefix = "X-"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setVersionHeaders(w.Header(), info, prefix)
			next.ServeHTTP(w, r)
		})
	}
}

// FiberMiddleware returns a Fiber middleware that adds version headers to all responses.
func FiberMiddleware(info *Info, prefix string) fiber.Handler {
	if info == nil {
		info = Default()
	}
	if prefix == "" {
		prefix = "X-"
	}

	return func(c fiber.Ctx) error {
		setVersionHeadersFiber(c, info, prefix)
		return c.Next()
	}
}

// TextHandler returns an http.HandlerFunc that serves version information as plain text.
func TextHandler(config ...HandlerConfig) http.HandlerFunc {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	cfg.HeaderPrefix = normalizeHeaderPrefix(cfg.HeaderPrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setVersionHeaders(w.Header(), cfg.payload(), cfg.HeaderPrefix)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cfg.textPayload()))
	}
}

// FiberTextHandler returns a Fiber handler that serves version information as plain text.
func FiberTextHandler(config ...HandlerConfig) fiber.Handler {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	cfg.HeaderPrefix = normalizeHeaderPrefix(cfg.HeaderPrefix)

	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setVersionHeadersFiber(c, cfg.payload(), cfg.HeaderPrefix)
		}

		return c.SendString(cfg.textPayload())
	}
}

// SimpleHandler returns a minimal handler that just returns the version string.
func SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(Default().String()))
	}
}

// FiberSimpleHandler returns a minimal Fiber handler that just returns the version string.
func FiberSimpleHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(Default().String())
	}
}

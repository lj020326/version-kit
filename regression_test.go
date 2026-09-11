package version

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

// TestHandlerOmitsBuildDetailsByDefault: the version endpoint is usually
// unauthenticated, and go_version lets anyone match a published Go runtime CVE
// to the exact build serving them. Commit, build date, platform and compiler
// narrow it further. None of it is a vulnerability on its own; it is
// reconnaissance that costs nothing to withhold.
func TestHandlerOmitsBuildDetailsByDefault(t *testing.T) {
	info := NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")

	rec := httptest.NewRecorder()
	Handler(HandlerConfig{Info: info})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	body, _ := io.ReadAll(rec.Result().Body)

	var parsed Info
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, body)
	}

	if parsed.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", parsed.Version)
	}
	if parsed.Branch != "main" {
		t.Errorf("branch = %q, want main", parsed.Branch)
	}
	for name, value := range map[string]string{
		"commit":     parsed.Commit,
		"build_date": parsed.BuildDate,
		"go_version": parsed.GoVersion,
		"platform":   parsed.Platform,
		"compiler":   parsed.Compiler,
	} {
		if value != "" {
			t.Errorf("%s = %q, want it withheld by default", name, value)
		}
	}
	if strings.Contains(string(body), runtime.Version()) {
		t.Errorf("the Go runtime version reached an unauthenticated response: %s", body)
	}
}

func TestHandlerIncludesBuildDetailsWhenAsked(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	rec := httptest.NewRecorder()
	Handler(HandlerConfig{Info: info, IncludeBuildDetails: true})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	body, _ := io.ReadAll(rec.Result().Body)

	var parsed Info
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Commit != "abc123" || parsed.GoVersion == "" {
		t.Errorf("IncludeBuildDetails did not take effect: %s", body)
	}
}

func TestTextHandlerOmitsBuildDetailsByDefault(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	rec := httptest.NewRecorder()
	TextHandler(HandlerConfig{Info: info})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	body, _ := io.ReadAll(rec.Result().Body)
	text := string(body)

	if !strings.Contains(text, "Version:") {
		t.Errorf("the version line is missing: %q", text)
	}
	for _, leaked := range []string{"abc123", runtime.Version(), "Go version:", "Compiler:"} {
		if strings.Contains(text, leaked) {
			t.Errorf("text response leaked %q: %q", leaked, text)
		}
	}
}

// TestInvalidHeaderPrefixFallsBack: the prefix is concatenated into a header
// NAME, so characters that cannot appear there produce a malformed header.
func TestInvalidHeaderPrefixFallsBack(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	for _, prefix := range []string{"", "bad prefix ", "x:y", "a\nb"} {
		rec := httptest.NewRecorder()
		Handler(HandlerConfig{
			Info:           info,
			IncludeHeaders: true,
			HeaderPrefix:   prefix,
		})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

		if got := rec.Header().Get("X-Version"); got != "1.0.0" {
			t.Errorf("prefix %q: X-Version = %q, want the fallback prefix to be used", prefix, got)
		}
	}

	// A valid prefix is honoured.
	rec := httptest.NewRecorder()
	Handler(HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-App-"})(
		rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if got := rec.Header().Get("X-App-Version"); got != "1.0.0" {
		t.Errorf("X-App-Version = %q, want 1.0.0", got)
	}
}

func TestPublicKeepsOnlySafeFields(t *testing.T) {
	full := NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")
	pub := full.Public()

	if pub.Version != "1.0.0" || pub.Branch != "main" {
		t.Errorf("Public() dropped a safe field: %+v", pub)
	}
	if pub.Commit != "" || pub.GoVersion != "" || pub.Platform != "" || pub.Compiler != "" || pub.BuildDate != "" {
		t.Errorf("Public() kept a build detail: %+v", pub)
	}
	// The original is untouched.
	if full.Commit != "abc123" {
		t.Error("Public() mutated its receiver")
	}
	if (*Info)(nil).Public() != nil {
		t.Error("Public() on nil should return nil")
	}
}

// --- Codex review follow-ups (PR #4) ---

// TestHeadersHonourIncludeBuildDetails is the regression test for headers being
// built from cfg.Info instead of the payload actually served: an endpoint with
// IncludeHeaders on but IncludeBuildDetails off still emitted the commit and
// build date, so the build fingerprint leaked through the header path.
func TestHeadersHonourIncludeBuildDetails(t *testing.T) {
	info := &Info{
		Version:   "1.2.3",
		Branch:    "main",
		Commit:    "abcdef1234567890",
		BuildDate: "2026-01-02T03:04:05Z",
		GoVersion: "go1.27.0",
	}

	for _, tc := range []struct {
		name    string
		handler func(HandlerConfig) http.HandlerFunc
	}{
		{"json", func(c HandlerConfig) http.HandlerFunc { return Handler(c) }},
		{"text", func(c HandlerConfig) http.HandlerFunc { return TextHandler(c) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.handler(HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-"})
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

			if got := rec.Header().Get("X-Commit"); got != "" {
				t.Errorf("X-Commit leaked without IncludeBuildDetails: %q", got)
			}
			if got := rec.Header().Get("X-Build-Date"); got != "" {
				t.Errorf("X-Build-Date leaked without IncludeBuildDetails: %q", got)
			}
			if got := rec.Header().Get("X-Version"); got != "1.2.3" {
				t.Errorf("X-Version = %q, want 1.2.3", got)
			}
			if got := rec.Header().Get("X-Branch"); got != "main" {
				t.Errorf("X-Branch = %q, want main", got)
			}
		})
	}

	t.Run("opted in", func(t *testing.T) {
		h := Handler(HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-", IncludeBuildDetails: true})
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

		if got := rec.Header().Get("X-Commit"); got != "abcdef1" {
			t.Errorf("X-Commit = %q, want abcdef1", got)
		}
		if got := rec.Header().Get("X-Build-Date"); got == "" {
			t.Error("X-Build-Date missing with IncludeBuildDetails")
		}
	})
}

// TestValidHeaderPrefixAcceptsTokenChars covers prefixes that are valid HTTP
// tokens but were rejected by the alphanumeric-and-dash-only check, which
// silently rewrote a caller's configured prefix to "X-".
func TestValidHeaderPrefixAcceptsTokenChars(t *testing.T) {
	valid := []string{"X-", "X.App-", "App_", "a~b+", "x^y-", "My!App-", "v1.2-"}
	for _, p := range valid {
		if !validHeaderPrefix(p) {
			t.Errorf("validHeaderPrefix(%q) = false, want true (all tchars)", p)
		}
	}

	invalid := []string{"", "X ", "X:", "X-\n", "X(", "X\"", "X@", "X/"}
	for _, p := range invalid {
		if validHeaderPrefix(p) {
			t.Errorf("validHeaderPrefix(%q) = true, want false (not a token)", p)
		}
	}

	// A valid, unusual prefix must survive into the emitted header name.
	h := Handler(HandlerConfig{Info: &Info{Version: "9.9.9"}, IncludeHeaders: true, HeaderPrefix: "X.App-"})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if got := rec.Header().Get("X.App-Version"); got != "9.9.9" {
		t.Errorf("X.App-Version = %q, want 9.9.9 (prefix was rewritten)", got)
	}
}

// TestTextHandlersNormalizeHeaderPrefix is the regression test for the text
// handlers skipping validHeaderPrefix: an invalid prefix reached the header
// setter instead of falling back to "X-" the way Handler documents.
func TestTextHandlersNormalizeHeaderPrefix(t *testing.T) {
	h := TextHandler(HandlerConfig{
		Info:           &Info{Version: "4.5.6"},
		IncludeHeaders: true,
		HeaderPrefix:   "bad prefix:",
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if got := rec.Header().Get("X-Version"); got != "4.5.6" {
		t.Errorf("X-Version = %q, want 4.5.6 (invalid prefix was not normalized)", got)
	}
	for name := range rec.Header() {
		if strings.Contains(name, " ") || strings.Contains(name, ":") {
			t.Errorf("malformed header name reached the response: %q", name)
		}
	}
}

// --- Codex review round 3 (PR #4) ---

// TestDocumentedBuildDetailsExampleCompiles pins the READMEs' build-details
// example to the actual API. Both guides called version.Get(), which does not
// exist -- the package-level constructor for ldflag-backed information is
// Default() -- so the one example a reader would copy to turn the full
// response back on did not compile.
func TestDocumentedBuildDetailsExampleCompiles(t *testing.T) {
	h := Handler(HandlerConfig{
		Info:                Default(),
		IncludeBuildDetails: true,
	})

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	// The documented point of the flag: the build fingerprint comes back.
	for _, field := range []string{"version", "commit", "build_date", "go_version"} {
		if _, ok := payload[field]; !ok {
			t.Errorf("field %q missing; IncludeBuildDetails must serve the full Info", field)
		}
	}

	// And the documented default keeps them out.
	def := Handler(HandlerConfig{Info: Default()})
	rec2 := httptest.NewRecorder()
	def(rec2, httptest.NewRequest(http.MethodGet, "/version", nil))

	var bare map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &bare); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"commit", "build_date", "go_version"} {
		if _, ok := bare[field]; ok {
			t.Errorf("field %q served by default; the READMEs promise only version and branch", field)
		}
	}
}

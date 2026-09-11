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

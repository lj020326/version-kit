// Package version provides application version information management.
// Includes version number, build time, Git commit hash, and HTTP endpoints
// for exposing version information in APIs.
package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// Default version variables - can be overridden via ldflags during build.
// Example: go build -ldflags "-X github.com/lj020326/version-kit/v2.Version=1.0.0"
var (
	// Version is the application version number
	Version = "dev"

	// Commit is the Git commit hash
	Commit = "unknown"

	// BuildDate is the build timestamp
	BuildDate = "unknown"

	// Branch is the Git branch name (optional)
	Branch = ""
)

// Info holds version information for an application.
type Info struct {
	// Version is the semantic version number (e.g., "1.2.3")
	Version string `json:"version"`

	// Commit is the Git commit hash (short or full)
	Commit string `json:"commit,omitempty"`

	// BuildDate is the build timestamp in RFC3339 format
	BuildDate string `json:"build_date,omitempty"`

	// Branch is the Git branch name (optional)
	Branch string `json:"branch,omitempty"`

	// GoVersion is the Go runtime version
	GoVersion string `json:"go_version,omitempty"`

	// Platform is the OS/Arch combination (e.g., "linux/amd64")
	Platform string `json:"platform,omitempty"`

	// Compiler is the Go compiler used
	Compiler string `json:"compiler,omitempty"`
}

// New creates a new Info with the provided values.
func New(version, commit, buildDate string) *Info {
	return &Info{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Compiler:  runtime.Compiler,
	}
}

// NewWithBranch creates a new Info with branch information.
func NewWithBranch(version, commit, buildDate, branch string) *Info {
	info := New(version, commit, buildDate)
	info.Branch = branch
	return info
}

// Default returns an Info using the package-level variables.
// This is useful when version info is set via ldflags.
func Default() *Info {
	return NewWithBranch(Version, Commit, BuildDate, Branch)
}

// Public returns a copy of i carrying only the fields that are safe to expose
// on an unauthenticated endpoint: the version, and the branch when one is set.
//
// GoVersion in particular lets anyone match a published Go runtime CVE to the
// exact build serving them, and Commit, BuildDate, Platform and Compiler
// narrow it further. None of that is a vulnerability on its own; it is
// reconnaissance that costs nothing to withhold.
func (i *Info) Public() *Info {
	if i == nil {
		return nil
	}
	return &Info{
		Version: i.Version,
		Branch:  i.Branch,
	}
}

// String returns a human-readable version string.
func (i *Info) String() string {
	if i.Commit != "" && i.Commit != "unknown" {
		shortCommit := i.Commit
		if len(shortCommit) > 7 {
			shortCommit = shortCommit[:7]
		}
		return fmt.Sprintf("%s (%s)", i.Version, shortCommit)
	}
	return i.Version
}

// Full returns a detailed version string with all information.
func (i *Info) Full() string {
	result := fmt.Sprintf("Version:    %s\n", i.Version)

	if i.Commit != "" && i.Commit != "unknown" {
		result += fmt.Sprintf("Commit:     %s\n", i.Commit)
	}

	if i.Branch != "" {
		result += fmt.Sprintf("Branch:     %s\n", i.Branch)
	}

	if i.BuildDate != "" && i.BuildDate != "unknown" {
		result += fmt.Sprintf("Built:      %s\n", i.BuildDate)
	}

	// Skip fields that were not populated, so a reduced Info (see Public)
	// renders without a run of empty labels.
	if i.GoVersion != "" {
		result += fmt.Sprintf("Go version: %s\n", i.GoVersion)
	}
	if i.Platform != "" {
		result += fmt.Sprintf("Platform:   %s\n", i.Platform)
	}
	if i.Compiler != "" {
		result += fmt.Sprintf("Compiler:   %s\n", i.Compiler)
	}

	return result
}

// JSON returns the version info as a JSON string.
func (i *Info) JSON() string {
	data, err := json.Marshal(i)
	if err != nil {
		return fmt.Sprintf(`{"version":"%s","error":"%s"}`, i.Version, err.Error())
	}
	return string(data)
}

// JSONPretty returns the version info as a pretty-printed JSON string.
func (i *Info) JSONPretty() string {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"version":"%s","error":"%s"}`, i.Version, err.Error())
	}
	return string(data)
}

// Map returns the version info as a map[string]string.
func (i *Info) Map() map[string]string {
	m := map[string]string{
		"version":    i.Version,
		"go_version": i.GoVersion,
		"platform":   i.Platform,
		"compiler":   i.Compiler,
	}

	if i.Commit != "" && i.Commit != "unknown" {
		m["commit"] = i.Commit
	}

	if i.Branch != "" {
		m["branch"] = i.Branch
	}

	if i.BuildDate != "" && i.BuildDate != "unknown" {
		m["build_date"] = i.BuildDate
	}

	return m
}

// Validate checks if the version info has valid required fields.
func (i *Info) Validate() error {
	if i.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

// IsDev returns true if this is a development version.
func (i *Info) IsDev() bool {
	return i.Version == "dev" || i.Version == "development" || i.Version == ""
}

// BuildTimestamp returns the build date as a time.Time.
// Returns zero time if parsing fails.
func (i *Info) BuildTimestamp() time.Time {
	if i.BuildDate == "" || i.BuildDate == "unknown" {
		return time.Time{}
	}

	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, i.BuildDate)
	if err == nil {
		return t
	}

	// Try common formats
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC1123,
		time.RFC1123Z,
	}

	for _, format := range formats {
		t, err = time.Parse(format, i.BuildDate)
		if err == nil {
			return t
		}
	}

	return time.Time{}
}

// ShortCommit returns the first 7 characters of the commit hash.
func (i *Info) ShortCommit() string {
	if i.Commit == "" || i.Commit == "unknown" {
		return ""
	}
	if len(i.Commit) > 7 {
		return i.Commit[:7]
	}
	return i.Commit
}

// Builder provides a fluent interface for creating Info.
type Builder struct {
	info *Info
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{
		info: &Info{
			GoVersion: runtime.Version(),
			Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			Compiler:  runtime.Compiler,
		},
	}
}

// WithVersion sets the version.
func (b *Builder) WithVersion(version string) *Builder {
	b.info.Version = version
	return b
}

// WithCommit sets the commit hash.
func (b *Builder) WithCommit(commit string) *Builder {
	b.info.Commit = commit
	return b
}

// WithBuildDate sets the build date.
func (b *Builder) WithBuildDate(buildDate string) *Builder {
	b.info.BuildDate = buildDate
	return b
}

// WithBranch sets the branch name.
func (b *Builder) WithBranch(branch string) *Builder {
	b.info.Branch = branch
	return b
}

// Build returns the constructed Info.
func (b *Builder) Build() *Info {
	return b.info
}

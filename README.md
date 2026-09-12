# Version Kit

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/version-kit/v2.svg)](https://pkg.go.dev/github.com/soulteary/version-kit/v2)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/version-kit/graph/badge.svg)](https://codecov.io/gh/soulteary/version-kit)

[中文文档](README_CN.md)

A version information management toolkit for Go applications. Provides structured version info, HTTP endpoints, and middleware for both net/http and Fiber.

## Features

- **Version Information**: Structured version info with version, commit, build date, branch, and runtime details
- **HTTP Endpoints**: JSON and text format endpoints for version APIs
- **Middleware**: Add version headers to all responses
- **Dual Framework Support**: Works with both net/http and Fiber
- **Builder Pattern**: Fluent interface for constructing version info
- **Build-time Injection**: Support for ldflags version injection

## Requirements

- **Go 1.26+** for building and running.
- Fiber APIs (`FiberHandler`, `FiberMiddleware`, etc.) require Fiber v3.4.0 or later.

This v2 module line targets Fiber v3. Applications that still use Fiber v2 should remain on `github.com/soulteary/version-kit` v1.

## Installation

```bash
go get github.com/soulteary/version-kit/v2
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    // Create version info
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    // Print version string
    fmt.Println(info.String()) // Output: 1.0.0 (abc123)
    
    // Print full version info
    fmt.Println(info.Full())
    
    // Get as JSON
    fmt.Println(info.JSON())
}
```

### Using Package Variables with ldflags

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    // Use default package variables
    // Set during build with ldflags
    info := version.Default()
    fmt.Println(info.String())
}
```

Build with version info:

```bash
go build -ldflags "\
  -X github.com/soulteary/version-kit/v2.Version=1.0.0 \
  -X github.com/soulteary/version-kit/v2.Commit=$(git rev-parse HEAD) \
  -X github.com/soulteary/version-kit/v2.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/soulteary/version-kit/v2.Branch=$(git rev-parse --abbrev-ref HEAD)" \
  -o myapp
```

For a short commit hash only, use `git rev-parse --short HEAD` instead of `git rev-parse HEAD` for the `Commit` variable.

### HTTP Endpoint (net/http)

```go
package main

import (
    "net/http"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    mux := http.NewServeMux()
    
    // Register JSON endpoint
    version.RegisterEndpoint(mux, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // Or use handler directly
    mux.HandleFunc("/v", version.Handler(version.HandlerConfig{Info: info}))
    
    // Text format endpoint
    mux.HandleFunc("/version.txt", version.TextHandler(version.HandlerConfig{Info: info}))
    
    // Simple version string
    mux.HandleFunc("/v/simple", version.SimpleHandler())
    
    http.ListenAndServe(":8080", mux)
}
```

### HTTP Endpoint (Fiber)

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    app := fiber.New()
    
    // Register JSON endpoint
    version.RegisterEndpointFiber(app, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // Or use handler directly
    app.Get("/v", version.FiberHandler(version.HandlerConfig{Info: info}))
    
    // Text format endpoint
    app.Get("/version.txt", version.FiberTextHandler(version.HandlerConfig{Info: info}))
    
    // Simple version string
    app.Get("/v/simple", version.FiberSimpleHandler())
    
    app.Listen(":3000")
}
```

### Version Headers Middleware

Add version information to all response headers:

```go
package main

import (
    "net/http"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello"))
    })
    
    // Wrap with version middleware
    wrapped := version.Middleware(info, "X-")(handler)
    
    // All responses will have:
    // X-Version: 1.0.0
    // X-Branch:  main     (when set)
    
    http.ListenAndServe(":8080", wrapped)
}
```

These headers ride on **every** response, so `Middleware` emits only the
public fields. `X-Commit` and `X-Build-Date` are an opt-in, exactly as they
are for the handlers:

```go
wrapped := version.MiddlewareWithConfig(version.HandlerConfig{
    Info:                info,
    HeaderPrefix:        "X-",
    IncludeBuildDetails: true, // adds X-Commit and X-Build-Date
})(handler)
```

Use it on internal routes, or behind authentication: the commit and build
date let anyone match a published CVE to the exact build serving them.

For Fiber:

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    app := fiber.New()
    
    // Add public version headers to all responses.
    // For X-Commit / X-Build-Date use:
    //   version.FiberMiddlewareWithConfig(version.HandlerConfig{
    //       Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true})
    app.Use(version.FiberMiddleware(info, "X-"))
    
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendString("Hello")
    })
    
    app.Listen(":3000")
}
```

### Builder Pattern

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.NewBuilder().
        WithVersion("1.0.0").
        WithCommit("abc123").
        WithBuildDate("2025-01-01T00:00:00Z").
        WithBranch("main").
        Build()
    
    fmt.Println(info.String())
}
```

### Include Headers in Endpoint Response

```go
version.RegisterEndpoint(mux, "/version", version.HandlerConfig{
    Info:           info,
    IncludeHeaders: true,  // Add version headers to response
    HeaderPrefix:   "X-App-", // Custom prefix
})
```

## API Reference

### Package-level functions

| Function | Description |
|----------|-------------|
| `New(version, commit, buildDate string) *Info` | Creates version info with the given version, commit, and build date. Runtime fields (Go version, platform, compiler) are set automatically. |
| `NewWithBranch(version, commit, buildDate, branch string) *Info` | Like `New` but also sets the branch name. |
| `Default() *Info` | Returns info from package variables (Version, Commit, BuildDate, Branch), typically set via ldflags. |
| `NewBuilder() *Builder` | Returns a builder for constructing `Info` with a fluent API. |
| `Middleware(info *Info, prefix string) func(http.Handler) http.Handler` | Adds the **public** version headers (`X-Version`, `X-Branch`) to every response. |
| `MiddlewareWithConfig(config HandlerConfig) func(http.Handler) http.Handler` | Same, with the full `HandlerConfig`; set `IncludeBuildDetails` for `X-Commit` and `X-Build-Date`. |
| `FiberMiddleware(info *Info, prefix string) fiber.Handler` | Fiber form of `Middleware`. |
| `FiberMiddlewareWithConfig(config HandlerConfig) fiber.Handler` | Fiber form of `MiddlewareWithConfig`. |

### Info Methods

| Method | Description |
|--------|-------------|
| `String()` | Returns version with short commit (e.g., "1.0.0 (abc1234)") |
| `Full()` | Returns detailed multi-line version info |
| `JSON()` | Returns JSON representation |
| `JSONPretty()` | Returns pretty-printed JSON |
| `Map()` | Returns version info as map[string]string |
| `Validate()` | Validates required fields |
| `IsDev()` | Returns true if version is "dev" or empty |
| `BuildTimestamp()` | Parses build date as time.Time |
| `ShortCommit()` | Returns first 7 characters of commit |

### HandlerConfig Options

Use `DefaultHandlerConfig()` to get a config with default values when you only want to override specific fields.

```go
type HandlerConfig struct {
    Info           *Info   // Version info (default: Default())
    Pretty              bool   // Pretty-print JSON (default: false)
    IncludeHeaders      bool   // Add version headers (default: false)
    HeaderPrefix        string // Header prefix (default: "X-")
    IncludeBuildDetails bool   // Serve the full Info (default: false)
}
```

### Package Variables

Set these via ldflags at build time:

```go
var (
    Version   = "dev"      // Application version
    Commit    = "unknown"  // Git commit hash
    BuildDate = "unknown"  // Build timestamp
    Branch    = ""         // Git branch name
)
```

## Example Response

These show the **full** response, which needs `IncludeBuildDetails: true`.
By default the endpoint serves only `version` and `branch` -- see
[Build Details](#build-details).

### JSON Endpoint

```json
{
  "version": "1.0.0",
  "commit": "abc123def456",
  "build_date": "2025-01-01T00:00:00Z",
  "branch": "main",
  "go_version": "go1.26",
  "platform": "linux/amd64",
  "compiler": "gc"
}
```

### Text Endpoint

```
Version:    1.0.0
Commit:     abc123def456
Branch:     main
Built:      2025-01-01T00:00:00Z
Go version: go1.26
Platform:   linux/amd64
Compiler:   gc
```

## Testing

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Build Details

`IncludeBuildDetails` is **false by default**, so the endpoint serves only
`version` and `branch`:

```json
{"version":"1.2.3","branch":"main"}
```

This endpoint is usually unauthenticated, and `go_version` lets anyone match a
published Go runtime CVE to the exact build serving them, while the commit and
build date fingerprint your deployment. Turn it on for an internal endpoint, or
behind authentication, to get the full response back:

```go
version.Handler(version.HandlerConfig{
    Info:                version.Default(),
    IncludeBuildDetails: true,
})
```

```json
{"version":"1.2.3","branch":"main","commit":"abc1234","build_date":"2026-01-02T03:04:05Z","go_version":"go1.27.0",...}
```

The same flag gates the `X-*-Commit` and `X-*-Build-Date` response headers, so
`IncludeHeaders` alone does not expose them.

## License

Apache License 2.0

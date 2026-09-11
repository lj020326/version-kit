# Version Kit

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/version-kit/v2.svg)](https://pkg.go.dev/github.com/soulteary/version-kit/v2)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/version-kit/graph/badge.svg)](https://codecov.io/gh/soulteary/version-kit)

[English](README.md)

一个用于 Go 应用程序的版本信息管理工具包。提供结构化的版本信息、HTTP 端点和中间件，同时支持 net/http 和 Fiber 框架。

## 功能特性

- **版本信息**: 结构化的版本信息，包含版本号、提交哈希、构建日期、分支和运行时详情
- **HTTP 端点**: JSON 和文本格式的版本 API 端点
- **中间件**: 为所有响应添加版本头信息
- **双框架支持**: 同时支持 net/http 和 Fiber
- **构建器模式**: 流式接口构建版本信息
- **构建时注入**: 支持通过 ldflags 注入版本信息

## 运行要求

- **Go 1.26+**，用于构建与运行。
- Fiber API（`FiberHandler`、`FiberMiddleware` 等）要求 Fiber v3.4.0 或更高版本。

此 v2 模块版本面向 Fiber v3。仍使用 Fiber v2 的应用应继续使用 `github.com/soulteary/version-kit` v1。

## 安装

```bash
go get github.com/soulteary/version-kit/v2
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    // 创建版本信息
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    // 打印版本字符串
    fmt.Println(info.String()) // 输出: 1.0.0 (abc123)
    
    // 打印完整版本信息
    fmt.Println(info.Full())
    
    // 获取 JSON 格式
    fmt.Println(info.JSON())
}
```

### 使用 ldflags 设置包变量

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    // 使用默认包变量
    // 在构建时通过 ldflags 设置
    info := version.Default()
    fmt.Println(info.String())
}
```

构建时注入版本信息:

```bash
go build -ldflags "\
  -X github.com/soulteary/version-kit/v2.Version=1.0.0 \
  -X github.com/soulteary/version-kit/v2.Commit=$(git rev-parse HEAD) \
  -X github.com/soulteary/version-kit/v2.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/soulteary/version-kit/v2.Branch=$(git rev-parse --abbrev-ref HEAD)" \
  -o myapp
```

若只需短提交哈希，可将 `Commit` 变量中的 `git rev-parse HEAD` 改为 `git rev-parse --short HEAD`。

### HTTP 端点 (net/http)

```go
package main

import (
    "net/http"
    
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    mux := http.NewServeMux()
    
    // 注册 JSON 端点
    version.RegisterEndpoint(mux, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // 或直接使用处理器
    mux.HandleFunc("/v", version.Handler(version.HandlerConfig{Info: info}))
    
    // 文本格式端点
    mux.HandleFunc("/version.txt", version.TextHandler(version.HandlerConfig{Info: info}))
    
    // 简单版本字符串
    mux.HandleFunc("/v/simple", version.SimpleHandler())
    
    http.ListenAndServe(":8080", mux)
}
```

### HTTP 端点 (Fiber)

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    app := fiber.New()
    
    // 注册 JSON 端点
    version.RegisterEndpointFiber(app, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // 或直接使用处理器
    app.Get("/v", version.FiberHandler(version.HandlerConfig{Info: info}))
    
    // 文本格式端点
    app.Get("/version.txt", version.FiberTextHandler(version.HandlerConfig{Info: info}))
    
    // 简单版本字符串
    app.Get("/v/simple", version.FiberSimpleHandler())
    
    app.Listen(":3000")
}
```

### 版本头信息中间件

为所有响应添加版本信息头:

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
    
    // 使用版本中间件包装
    wrapped := version.Middleware(info, "X-")(handler)
    
    // 所有响应将包含:
    // X-Version: 1.0.0
    // X-Commit: abc123
    // X-Build-Date: 2025-01-01T00:00:00Z
    
    http.ListenAndServe(":8080", wrapped)
}
```

Fiber 版本:

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    version "github.com/soulteary/version-kit/v2"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    app := fiber.New()
    
    // 为所有响应添加版本头信息
    app.Use(version.FiberMiddleware(info, "X-"))
    
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendString("Hello")
    })
    
    app.Listen(":3000")
}
```

### 构建器模式

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

### 在端点响应中包含头信息

```go
version.RegisterEndpoint(mux, "/version", version.HandlerConfig{
    Info:           info,
    IncludeHeaders: true,     // 在响应中添加版本头信息
    HeaderPrefix:   "X-App-", // 自定义前缀
})
```

## API 参考

### 包级函数

| 函数 | 描述 |
|------|------|
| `New(version, commit, buildDate string) *Info` | 根据版本号、提交哈希和构建日期创建版本信息，运行时字段（Go 版本、平台、编译器）会自动填充。 |
| `NewWithBranch(version, commit, buildDate, branch string) *Info` | 与 `New` 类似，同时设置分支名。 |
| `Default() *Info` | 使用包变量（Version、Commit、BuildDate、Branch）构造信息，通常由 ldflags 在构建时注入。 |
| `NewBuilder() *Builder` | 返回用于以流式 API 构建 `Info` 的 Builder。 |

### Info 方法

| 方法 | 描述 |
|------|------|
| `String()` | 返回带短提交哈希的版本 (例如 "1.0.0 (abc1234)") |
| `Full()` | 返回详细的多行版本信息 |
| `JSON()` | 返回 JSON 表示 |
| `JSONPretty()` | 返回格式化的 JSON |
| `Map()` | 返回 map[string]string 格式的版本信息 |
| `Validate()` | 验证必填字段 |
| `IsDev()` | 如果版本是 "dev" 或空则返回 true |
| `BuildTimestamp()` | 解析构建日期为 time.Time |
| `ShortCommit()` | 返回提交哈希的前 7 个字符 |

### HandlerConfig 选项

仅需覆盖部分字段时，可使用 `DefaultHandlerConfig()` 获取默认配置后再修改。

```go
type HandlerConfig struct {
    Info                *Info  // 版本信息 (默认: Default())
    Pretty              bool   // 格式化 JSON 输出 (默认: false)
    IncludeHeaders      bool   // 添加版本头信息 (默认: false)
    HeaderPrefix        string // 头信息前缀 (默认: "X-")
    IncludeBuildDetails bool   // 返回完整 Info (默认: false)
}
```

### 包变量

在构建时通过 ldflags 设置这些变量:

```go
var (
    Version   = "dev"      // 应用版本
    Commit    = "unknown"  // Git 提交哈希
    BuildDate = "unknown"  // 构建时间戳
    Branch    = ""         // Git 分支名
)
```

## 响应示例

以下是**完整**响应，需要设置 `IncludeBuildDetails: true`。
默认情况下端点只返回 `version` 和 `branch`，详见[构建详情](#构建详情)。

### JSON 端点

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

### 文本端点

```
Version:    1.0.0
Commit:     abc123def456
Branch:     main
Built:      2025-01-01T00:00:00Z
Go version: go1.26
Platform:   linux/amd64
Compiler:   gc
```

## 构建详情

`IncludeBuildDetails` **默认为 false**，因此端点只返回 `version` 和 `branch`：

```json
{"version":"1.2.3","branch":"main"}
```

这个端点通常是不需要认证的，而 `go_version` 会让任何人把已公开的 Go 运行时 CVE
精确对应到正在服务的这个构建上，commit 与构建时间则等于给部署留下指纹。
仅在内部端点、或已有认证保护的端点上开启它，以取回完整响应：

```go
version.Handler(version.HandlerConfig{
    Info:                version.Default(),
    IncludeBuildDetails: true,
})
```

```json
{"version":"1.2.3","branch":"main","commit":"abc1234","build_date":"2026-01-02T03:04:05Z","go_version":"go1.27.0",...}
```

同一个开关也控制 `X-*-Commit` 与 `X-*-Build-Date` 响应头，因此单独开启
`IncludeHeaders` 并不会暴露它们。

## 测试

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 许可证

Apache License 2.0

### 构建详情

`IncludeBuildDetails` **默认为 false**，接口只返回 `version` 与 `branch`：

```json
{"version":"1.2.3","branch":"main"}
```

该接口通常无需认证，而 `go_version` 会让任何人把已公开的 Go 运行时 CVE 精确对应到
正在服务的构建版本，commit 与构建时间则会暴露部署指纹。若需要完整响应，请在内网接口
或认证之后开启：

```go
version.Handler(version.HandlerConfig{
    Info:                version.Default(),
    IncludeBuildDetails: true,
})
```

同一开关也控制 `X-*-Commit` 与 `X-*-Build-Date` 响应头，因此仅开启 `IncludeHeaders`
不会暴露它们。


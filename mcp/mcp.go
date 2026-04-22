// Package mcp exposes the x-viral-go [viral.Scorer] surface as a set of MCP
// (Model Context Protocol) tools that any host application can mount on its
// own MCP server.
//
// All tools wrap exported methods on *viral.Scorer. Each tool is defined via
// [mcptool.Define] so the JSON input schema is reflected from the typed input
// struct — no hand-maintained schemas, no drift.
//
// Usage from a host application:
//
//	import (
//	    "github.com/teslashibe/mcptool"
//	    viral "github.com/teslashibe/x-viral-go"
//	    xvmcp "github.com/teslashibe/x-viral-go/mcp"
//	)
//
//	scorer := viral.New(viral.WithLLMProvider(myProvider))
//	for _, tool := range xvmcp.Provider{}.Tools() {
//	    // register tool with your MCP server, passing scorer as the client arg
//	    // when invoking
//	}
//
// The platform identifier is "xviral" (rather than "x") to keep this package
// distinct from the x-go scraper, which owns the "x" / "x_" namespace.
//
// The [Excluded] map documents methods on *Scorer that are intentionally not
// exposed via MCP, with a one-line reason. The coverage test in mcp_test.go
// fails if a new exported method is added without either being wrapped by a
// tool or appearing in [Excluded].
package mcp

import "github.com/teslashibe/mcptool"

// Provider implements [mcptool.Provider] for x-viral-go. The zero value is
// ready to use.
type Provider struct{}

// Platform returns "xviral".
func (Provider) Platform() string { return "xviral" }

// Tools returns every x-viral-go MCP tool, in registration order.
func (Provider) Tools() []mcptool.Tool {
	out := make([]mcptool.Tool, 0, len(scoreTools)+len(optimizeTools)+len(weightsTools))
	out = append(out, scoreTools...)
	out = append(out, optimizeTools...)
	out = append(out, weightsTools...)
	return out
}

package mcp_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/teslashibe/mcptool"
	viral "github.com/teslashibe/x-viral-go"
	xvmcp "github.com/teslashibe/x-viral-go/mcp"
)

// TestEveryScorerMethodIsWrappedOrExcluded fails when a new exported method
// is added to *viral.Scorer without either being wrapped by an MCP tool or
// being added to xvmcp.Excluded with a reason. This is the drift-prevention
// mechanism: keeping the MCP surface in lockstep with the package API is
// enforced by CI rather than convention.
func TestEveryScorerMethodIsWrappedOrExcluded(t *testing.T) {
	rep := mcptool.Coverage(
		reflect.TypeOf(&viral.Scorer{}),
		xvmcp.Provider{}.Tools(),
		xvmcp.Excluded,
	)
	if len(rep.Missing) > 0 {
		t.Fatalf("methods missing MCP exposure (add a tool or list in excluded.go): %v", rep.Missing)
	}
	if len(rep.UnknownExclusions) > 0 {
		t.Fatalf("excluded.go references methods that don't exist on *Scorer (rename?): %v", rep.UnknownExclusions)
	}
	if len(rep.Wrapped)+len(rep.Excluded) == 0 {
		t.Fatal("no wrapped or excluded methods detected — coverage helper is mis-configured")
	}
}

// TestToolsValidate verifies every tool has a non-empty name in canonical
// snake_case form, a description within length limits, and a non-nil Invoke
// + InputSchema.
func TestToolsValidate(t *testing.T) {
	if err := mcptool.ValidateTools(xvmcp.Provider{}.Tools()); err != nil {
		t.Fatal(err)
	}
}

// TestPlatformName guards against accidental rebrands. "xviral" is the
// chosen identifier so the package does not collide with x-go's "x" / "x_"
// namespace.
func TestPlatformName(t *testing.T) {
	if got := (xvmcp.Provider{}).Platform(); got != "xviral" {
		t.Errorf("Platform() = %q, want xviral", got)
	}
}

// TestToolsHaveXviralPrefix encodes the per-platform naming convention.
func TestToolsHaveXviralPrefix(t *testing.T) {
	const prefix = "xviral_"
	for _, tool := range (xvmcp.Provider{}).Tools() {
		if !strings.HasPrefix(tool.Name, prefix) {
			t.Errorf("tool %q lacks %s prefix", tool.Name, prefix)
		}
	}
}

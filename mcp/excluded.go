package mcp

// Excluded enumerates exported methods on *viral.Scorer that are intentionally
// not exposed via MCP. Each entry must have a non-empty reason.
//
// The coverage test in mcp_test.go fails if any exported method on *Scorer is
// neither wrapped by a Tool nor present in this map (or vice-versa: if an
// entry here doesn't correspond to a real method).
//
// When the underlying scorer gains a new method:
//   - prefer to add an MCP tool for it (see score.go / optimize.go / weights.go)
//   - if the method is unsuitable for an agent (host-side wiring helper,
//     internal-only observability, etc.), add it here with a reason
var Excluded = map[string]string{
	// All current *viral.Scorer methods are wrapped by tools.
}

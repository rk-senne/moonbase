package pipeline

import (
	"encoding/json"
	"strings"
)

// PhaseMeta is the structured metadata block agents can emit at the end of their
// output. When present, it provides reliable structured data for pipeline routing
// instead of relying on regex heuristics.
//
// Agents emit it as a JSON block:
//
//	{"__moonbase_meta": {
//	  "files_changed": ["src/foo.ts", "src/bar.ts"],
//	  "risk": "LOW",
//	  "ac_results": {"AC-1": "PASS", "AC-2": "PASS"},
//	  "decisions": ["Used existing pattern from auth.routes.ts"]
//	}}
type PhaseMeta struct {
	FilesChanged []string          `json:"files_changed,omitempty"`
	Risk         string            `json:"risk,omitempty"`
	ACResults    map[string]string `json:"ac_results,omitempty"`
	Decisions    []string          `json:"decisions,omitempty"`
}

type metaEnvelope struct {
	Meta PhaseMeta `json:"__moonbase_meta"`
}

// ParseMeta extracts the structured __moonbase_meta JSON block from agent output.
// Returns nil if no valid meta block is found. The parser searches for the last
// occurrence of `{"__moonbase_meta":` in the output to handle cases where the
// agent includes other JSON in its response.
func ParseMeta(output string) *PhaseMeta {
	// Find the last occurrence of the meta marker
	marker := `{"__moonbase_meta"`
	idx := strings.LastIndex(output, marker)
	if idx == -1 {
		return nil
	}

	// Extract from marker to the end, find the matching closing brace
	candidate := output[idx:]
	// Find the balanced closing brace
	depth := 0
	endIdx := -1
	for i, ch := range candidate {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx == -1 {
		return nil
	}

	jsonStr := candidate[:endIdx]

	var envelope metaEnvelope
	if err := json.Unmarshal([]byte(jsonStr), &envelope); err != nil {
		return nil
	}

	return &envelope.Meta
}

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

// Context injection and pricing resolution for missions.

// injectFileContext reads files mentioned in the Architecture output and injects
// their contents into the prompt. Enhancement 3: Pre-flight file injection.
func injectFileContext(pCtx *pipeline.PipelineContext) string {
	if len(pCtx.FilesChanged) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- PRE-FLIGHT FILE CONTEXT ---\n")
	sb.WriteString("These files were identified in the design phase. Current contents:\n\n")

	totalSize := 0
	const maxFileSize = 8000
	const maxTotalSize = 32000

	for _, f := range pCtx.FilesChanged {
		if totalSize >= maxTotalSize {
			sb.WriteString("\n...(remaining files omitted for context budget)\n")
			break
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > maxFileSize {
			content = content[:maxFileSize] + "\n...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f, content))
		totalSize += len(content)
	}

	sb.WriteString("--- END PRE-FLIGHT FILE CONTEXT ---\n")
	return sb.String()
}

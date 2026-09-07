package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// Context injection and pricing resolution for missions.

// applyPipelineConfig copies configuration onto a freshly constructed pipeline.
//
// pipeline.New and friends set hardcoded defaults, and nothing previously copied
// config over them, so several documented keys were inert: phase_timeout_seconds,
// max_output_size and max_retries were all settable and described in Config but
// never read. Raising phase_timeout_seconds had no effect, and --trace printed the
// hardcoded default, which made it look as though the setting was honoured.
//
// Zero values are ignored so an unset key keeps the pipeline default rather than
// clamping it to zero — a zero PhaseTimeout would fail every phase instantly.
func applyPipelineConfig(p *pipeline.Pipeline, cfg config.Config, sequential bool) {
	if cfg.PhaseTimeout > 0 {
		p.PhaseTimeout = time.Duration(cfg.PhaseTimeout) * time.Second
	}
	if cfg.MaxOutputSize > 0 {
		p.MaxOutputSize = cfg.MaxOutputSize
	}
	if cfg.MaxRetries > 0 {
		p.MaxRetries = cfg.MaxRetries
	}

	p.ParallelSpecialists = cfg.ParallelSpecialists
	if cfg.MaxSpecialistConcurrency > 0 {
		p.MaxSpecialistConcurrency = cfg.MaxSpecialistConcurrency
	}
	if sequential {
		p.ParallelSpecialists = false
	}
}

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

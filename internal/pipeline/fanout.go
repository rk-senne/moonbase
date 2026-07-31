// Fan-out orchestrator for parallel specialist execution.
//
// After QA (Phase 4) returns RiskLow, independent conditional specialists can
// execute concurrently rather than sequentially. This file provides the bounded-
// concurrency fan-out using sync.WaitGroup + a buffered-channel semaphore.
//
// Key design decisions:
//   - WaitGroup+semaphore (not errgroup) for partial-failure tolerance: one
//     specialist failing does NOT cancel siblings.
//   - Results sorted by phase number post-collection for deterministic output.
//   - Independence classified from agent metadata (shell.read_only, tools list).
package pipeline

import (
	"context"
	"sort"
	"sync"
	"time"
)

// FanOutResult captures the outcome of a single specialist execution.
type FanOutResult struct {
	Phase    int           // Phase number of the specialist
	Agent    string        // Agent name (e.g., "numbuh-274")
	Output   string        // Agent output (empty on failure)
	Err      error         // Non-nil if specialist failed
	Duration time.Duration // Wall-clock execution time
	Status   PhaseStatus   // StatusComplete or StatusFailed
}

// FanOutConfig controls parallel specialist execution.
type FanOutConfig struct {
	MaxConcurrency int           // Semaphore size (default 4, clamped 1–16)
	PhaseTimeout   time.Duration // Per-specialist timeout (0 = no timeout)
	TraceID        string        // For flywheel correlation
}

// SpecialistFunc is the execution function for a single specialist.
// The implementation wraps the backend deployment call.
type SpecialistFunc func(ctx context.Context, phase Phase) (string, error)

// RunSpecialists executes triggered specialists concurrently with bounded parallelism.
// Returns results sorted by phase number for deterministic aggregation.
// Partial failures do NOT cancel other specialists.
func RunSpecialists(
	ctx context.Context,
	phases []Phase,
	execute SpecialistFunc,
	cfg FanOutConfig,
) []FanOutResult {
	if len(phases) == 0 {
		return nil
	}

	maxConc := clampConcurrency(cfg.MaxConcurrency)

	// Semaphore: buffered channel limits active goroutines.
	sem := make(chan struct{}, maxConc)

	var mu sync.Mutex
	results := make([]FanOutResult, 0, len(phases))

	var wg sync.WaitGroup
	for _, phase := range phases {
		wg.Add(1)
		go func(p Phase) {
			defer wg.Done()

			// Acquire semaphore slot (or bail on cancellation).
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				results = append(results, FanOutResult{
					Phase:  p.Number,
					Agent:  p.AgentName,
					Err:    ctx.Err(),
					Status: StatusFailed,
				})
				mu.Unlock()
				return
			}

			start := time.Now()

			// Apply per-specialist timeout if configured.
			var execCtx context.Context
			var cancel context.CancelFunc
			if cfg.PhaseTimeout > 0 {
				execCtx, cancel = context.WithTimeout(ctx, cfg.PhaseTimeout)
			} else {
				execCtx, cancel = context.WithCancel(ctx)
			}

			output, err := execute(execCtx, p)
			cancel()
			elapsed := time.Since(start)

			result := FanOutResult{
				Phase:    p.Number,
				Agent:    p.AgentName,
				Output:   output,
				Duration: elapsed,
			}
			if err != nil {
				result.Err = err
				result.Status = StatusFailed
			} else {
				result.Status = StatusComplete
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(phase)
	}

	wg.Wait()

	// Sort by phase number for deterministic ordering.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Phase < results[j].Phase
	})

	return results
}

// TriggeredSpecialists evaluates which conditional specialists should fire
// based on trigger conditions and returns only those that pass.
// Reuses the existing EvaluateTrigger mechanism for consistency.
func TriggeredSpecialists(phases []Phase, pctx *PipelineContext) []Phase {
	var triggered []Phase
	for _, p := range phases {
		if !p.Conditional {
			continue
		}
		result := EvaluateTrigger(p.TriggerSpec, pctx)
		if result.Invoke {
			triggered = append(triggered, p)
		}
	}
	return triggered
}

// IsIndependentSpecialist determines if a specialist agent is safe for parallel
// execution. Independence requires:
//   - The phase is conditional (specialist, not core pipeline)
//   - The agent has shell.read_only == true OR does not have "write" in its tools
//
// This is computed from agent metadata, never hardcoded.
func IsIndependentSpecialist(phase Phase, tools []string, shellReadOnly *bool) bool {
	if !phase.Conditional {
		return false
	}

	// If shell config has read_only: true, the agent is independent.
	if shellReadOnly != nil && *shellReadOnly {
		return true
	}

	// If "write" is not in the tools list, the agent is independent.
	for _, t := range tools {
		if t == "write" {
			return false
		}
	}
	return true
}

// clampConcurrency constrains the concurrency value to the valid range [1, 16].
func clampConcurrency(n int) int {
	if n < 1 {
		return 4 // default
	}
	if n > 16 {
		return 16
	}
	return n
}

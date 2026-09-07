package main

import (
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// phase_timeout_seconds, max_output_size and max_retries were documented in
// Config and settable in config.yaml, but nothing copied them onto the pipeline —
// pipeline.New's hardcoded defaults always won. Raising phase_timeout_seconds had
// no effect, and --trace printed the hardcoded value, which made it look as though
// the setting had been honoured.

func TestApplyPipelineConfig_AppliesTimeoutOutputAndRetries(t *testing.T) {
	p := pipeline.New("task")
	cfg := config.Config{
		PhaseTimeout:             1500,
		MaxOutputSize:            250000,
		MaxRetries:               7,
		ParallelSpecialists:      true,
		MaxSpecialistConcurrency: 6,
	}

	applyPipelineConfig(p, cfg, false)

	if want := 1500 * time.Second; p.PhaseTimeout != want {
		t.Errorf("PhaseTimeout = %s, want %s", p.PhaseTimeout, want)
	}
	if p.MaxOutputSize != 250000 {
		t.Errorf("MaxOutputSize = %d, want 250000", p.MaxOutputSize)
	}
	if p.MaxRetries != 7 {
		t.Errorf("MaxRetries = %d, want 7", p.MaxRetries)
	}
	if p.MaxSpecialistConcurrency != 6 {
		t.Errorf("MaxSpecialistConcurrency = %d, want 6", p.MaxSpecialistConcurrency)
	}
}

// An unset key must leave the pipeline default intact rather than clamping to
// zero — a zero PhaseTimeout would make every phase fail instantly.
func TestApplyPipelineConfig_ZeroValuesKeepDefaults(t *testing.T) {
	p := pipeline.New("task")
	defaultTimeout := p.PhaseTimeout
	defaultOutput := p.MaxOutputSize
	defaultRetries := p.MaxRetries
	defaultConcurrency := p.MaxSpecialistConcurrency

	applyPipelineConfig(p, config.Config{ParallelSpecialists: true}, false)

	if p.PhaseTimeout != defaultTimeout {
		t.Errorf("PhaseTimeout = %s, want the default %s preserved", p.PhaseTimeout, defaultTimeout)
	}
	if p.MaxOutputSize != defaultOutput {
		t.Errorf("MaxOutputSize = %d, want default %d", p.MaxOutputSize, defaultOutput)
	}
	if p.MaxRetries != defaultRetries {
		t.Errorf("MaxRetries = %d, want default %d", p.MaxRetries, defaultRetries)
	}
	if p.MaxSpecialistConcurrency != defaultConcurrency {
		t.Errorf("MaxSpecialistConcurrency = %d, want default %d",
			p.MaxSpecialistConcurrency, defaultConcurrency)
	}
}

func TestApplyPipelineConfig_SequentialOverridesParallel(t *testing.T) {
	p := pipeline.New("task")
	cfg := config.Config{ParallelSpecialists: true, MaxSpecialistConcurrency: 4}

	applyPipelineConfig(p, cfg, true)

	if p.ParallelSpecialists {
		t.Error("--sequential must disable parallel specialists even when config enables them")
	}
}

func TestApplyPipelineConfig_ConfigCanDisableParallel(t *testing.T) {
	p := pipeline.New("task")

	applyPipelineConfig(p, config.Config{ParallelSpecialists: false}, false)

	if p.ParallelSpecialists {
		t.Error("parallel_specialists: false must be honoured")
	}
}

// The raised default is deliberate: phase timeouts were unenforceable until the
// backend became cancellable, and a measured implementation phase took 5m43s, so
// the old 5-minute default would now cause spurious cancellations.
func TestPipelineDefault_PhaseTimeoutIsRealistic(t *testing.T) {
	p := pipeline.New("task")

	if p.PhaseTimeout < 10*time.Minute {
		t.Errorf("default PhaseTimeout = %s; too low for a real implementation phase "+
			"(measured 5m43s) now that timeouts are enforced", p.PhaseTimeout)
	}
}

# Configuration

Moonbase reads YAML config from `~/.config/moonbase/config.yaml`. View the effective config with `moonbase config`. **No secrets live in config** — API keys come from environment variables only.

## Full reference

```yaml
# --- Core ---
default_backend: kiro-cli        # Preferred AI backend
theme: moonbase                  # TUI theme: moonbase | treehouse | classified | nerv
agents_dir: ""                   # Custom agents dir (empty = auto-detect)
agent_order: []                  # Display order for agents in the TUI sidebar

# --- Backend / execution ---
trust_tools: false               # Pass --trust-all-tools to kiro-cli (headless execution)
pipeline_backend: ""             # Backend for analysis phases (anthropic/openai); kiro-cli for implementation
fast_threshold: 0                # Word count below which --fast auto-engages (0 = disabled)
phase_timeout_seconds: 300       # Max seconds per phase
max_output_size: 100000          # Max output bytes per phase
max_retries: 1                   # Retries per phase before failure
enable_trace: false              # Generate trace IDs for pipeline runs
use_cmux: false                  # Auto-enable cmux features (notifications, split panes)

# --- Parallel specialists (see The Pipeline) ---
parallel_specialists: true       # Fan out independent specialists concurrently
max_specialist_concurrency: 4    # Concurrency cap (range 1–16; 1 = sequential)

# --- Kiro-native interop (see Kiro-Native-Interop) ---
compile:
  out_dir: .kiro/agents          # Where moonbase compile writes JSON
  auto_validate: true            # Run kiro-cli agent validate after compile
deploy:
  mode: legacy                   # legacy (raw prompt) | native (compiled JSON)
  auto_compile: false            # Recompile stale agents on deploy
safety:
  delegate_to_kiro: false        # When true + native mode: skip moonbase safety checks

# --- Token / cost observability (see Flywheel-and-Observability) ---
token_budget:
  max_tokens_per_mission: 0      # Hard cap (0 = unlimited)
  warn_threshold_pct: 80         # Warn at this % of budget
model_pricing:                   # Override default per-model prices (USD per 1M tokens)
  gpt-4o:
    prompt: 2.50
    completion: 10.00
```

## Notes

- **Backends**: `kiro-cli` (primary), OpenAI, Anthropic, Kimi, Ollama, and a clipboard fallback. API backends read keys from the environment (e.g., `OPENAI_API_KEY`).
- **Security defaults**: SafeEnv passes only allowlisted env vars to child processes; the hook guard blocks dangerous shell commands; user data is written `0600`/`0700`.
- **Adaptive depth** has no config — it's the default behavior of `moonbase mission`; use `--fast`/`--full`/`--depth` to override per-run.

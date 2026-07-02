# Design: Moonbase v2

## Architecture Decision

The Go TUI reads agent `.md` files by splitting on YAML frontmatter delimiters (`---`), parsing the top section as YAML (metadata), and using the rest as the agent's system prompt. The pipeline becomes a real orchestration engine that chains agent invocations through the Kiro CLI backend with context accumulation.

**Key ADR:** Kiro CLI is the primary backend. Other backends (OpenAI API, Anthropic API, Ollama) remain supported but secondary. The pipeline executes by spawning `kiro-cli chat` sessions with agent prompts — the same way the `subagent` tool works in this environment.

---

## Files Affected

| File | Change Type | Purpose |
|------|------------|---------|
| `internal/agents/agent.go` | modify | New Agent struct matching YAML frontmatter fields |
| `internal/agents/registry.go` | modify | Load `.md` files, parse YAML frontmatter |
| `internal/agents/parser.go` | new | YAML frontmatter extraction + parsing |
| `internal/pipeline/pipeline.go` | modify | Real phase execution with backend invocation |
| `internal/pipeline/context.go` | new | Context accumulation between phases |
| `internal/pipeline/riskgate.go` | new | Risk-gate routing logic (parse QA output) |
| `internal/backend/backend.go` | modify | Add context/prompt composition methods |
| `internal/backend/kiro.go` | modify | Real kiro-cli invocation with agent prompt |
| `internal/discovery/discovery.go` | new | Project context discovery (.kiro/specs, steering, build config) |
| `internal/discovery/steering.go` | new | Steering file loading with inclusion filtering |
| `cmd/moonbase/main.go` | modify | Add `install` subcommand |
| `cmd/moonbase/install.go` | new | Install agents into project .kiro/agents/ |
| `go.mod` | modify | Add `gopkg.in/yaml.v3` dependency |

---

## Component Interfaces

### Agent Struct (new)

```go
type Agent struct {
    // Frontmatter fields
    Name            string              `yaml:"name"`
    Designation     string              `yaml:"designation"`
    Role            string              `yaml:"role"`
    Description     string              `yaml:"description"`
    Tools           []string            `yaml:"tools"`
    AutoTools       []string            `yaml:"auto_tools"`
    Shell           *ShellConfig        `yaml:"shell,omitempty"`
    Write           *WriteConfig        `yaml:"write,omitempty"`
    Routing         *RoutingConfig      `yaml:"routing,omitempty"`
    Hooks           *HooksConfig        `yaml:"hooks,omitempty"`
    PipelinePosition *int               `yaml:"pipeline_position,omitempty"`
    Shortcut        string              `yaml:"shortcut"`
    Triggers        *string             `yaml:"triggers,omitempty"`

    // Derived from markdown body
    Prompt          string              `yaml:"-"`  // Everything after frontmatter
    FilePath        string              `yaml:"-"`  // Source file path
}

type ShellConfig struct {
    AllowedCommands []string `yaml:"allowed_commands"`
    ReadOnly        bool     `yaml:"read_only"`
}

type WriteConfig struct {
    Auto            []string `yaml:"auto"`
    Denied          []string `yaml:"denied"`
    RequiresApproval []string `yaml:"requires_approval"`
}

type RoutingConfig struct {
    Available []string `yaml:"available"`
    Trusted   []string `yaml:"trusted"`
}

type HooksConfig struct {
    OnActivate []Hook `yaml:"on_activate"`
}

type Hook struct {
    Command   string `yaml:"command"`
    TimeoutMs int    `yaml:"timeout_ms"`
}
```

### Parser Interface

```go
// ParseAgentFile reads a .md file and returns frontmatter + body
func ParseAgentFile(path string) (*Agent, error)

// SplitFrontmatter splits "---\nyaml\n---\nmarkdown" into (yaml, markdown)
func SplitFrontmatter(content []byte) ([]byte, []byte, error)
```

### Discovery Interface

```go
type ProjectContext struct {
    Specs      []SpecFile      // .kiro/specs/*/{requirements,design,tasks}.md
    Steering   []SteeringFile  // .kiro/steering/*.md (respecting inclusion)
    Stack      StackInfo       // Detected from build configs
    README     string          // README.md content (truncated)
}

func Discover(projectDir string) (*ProjectContext, error)
```

### Pipeline Context

```go
type PipelineContext struct {
    Task           string
    PhaseOutputs   map[int]string  // phase number → output
    RiskLevel      string          // current risk assessment
    FilesChanged   []string        // accumulated files touched
    Decisions      []string        // decisions recorded during pipeline
}

func (pc *PipelineContext) ForPhase(phase int) string  // Compose input for next phase
```

---

## Data Flow

### Agent Loading

```
agents/*.md → SplitFrontmatter() → YAML parse (metadata) + body (prompt)
                                          ↓
                                    Agent struct populated
                                          ↓
                                    Registry.agents []Agent
```

### Agent Deployment

```
User selects agent
        ↓
Discovery.Discover(cwd) → ProjectContext
        ↓
Compose prompt = ProjectContext.Steering + Agent.Prompt + ProjectContext.Specs
        ↓
Backend.Deploy(composedPrompt, task)
        ↓
kiro-cli chat --system-prompt <composed> --task <user_task>
```

### Pipeline Execution

```
User provides task
        ↓
Pipeline.New(task)
        ↓
For each mandatory phase:
    ├── Get agent by pipeline_position
    ├── Compose prompt (agent + project context + accumulated pipeline context)
    ├── Backend.Deploy(agent, phase_input)
    ├── Capture output
    ├── PipelineContext.PhaseOutputs[phase] = output
    ├── If phase == 4 (QA): parse risk level
    │   ├── LOW → continue to phase 5
    │   ├── MEDIUM → loop back to phase 3
    │   ├── HIGH → loop back to phase 2
    │   └── CRITICAL → stop, escalate
    └── If conditional phase: evaluate triggers
```

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| `.md` file with no frontmatter | Skip with warning, not a valid agent |
| Malformed YAML in frontmatter | Skip agent, log parse error |
| Agent with no `pipeline_position` | Valid (specialist) — available for direct deploy, not in pipeline |
| Empty `tools` list | Agent gets no tool access — valid for read-only roles |
| Pipeline context exceeds token limit | Summarize earlier phases, keep latest 2 in full |
| QA output doesn't contain risk level | Default to MEDIUM (cautious), warn user |
| Steering file with `inclusion: manual` | Check frontmatter for `inclusion` field; skip if manual |
| No `.kiro/` directory in project | Agent runs without project context (still functional) |
| Multiple specs directories | Show most recent or ask user which feature |

---

## Error Handling

| Error | Response | Requirement |
|-------|----------|-------------|
| No agents found in dir | Fatal error with helpful message | AC-1.1 |
| YAML parse failure | Skip agent, log error, continue loading others | AC-1.1 |
| Backend not available | Fall back to clipboard copy | AC-2.1 |
| Pipeline phase timeout | Mark phase failed, offer retry/skip | AC-2.1 |
| Risk gate infinite loop | Max 2 loops enforced, escalate to human | AC-2.2 |
| Discovery IO error | Continue without project context, log warning | AC-3.1 |
| Install target exists | Prompt overwrite or skip | AC-4.1 |

---

## Security Considerations
- Steering file content may contain project secrets referenced by name — never expand env vars in steering injection
- Agent prompts should never include raw secret values
- Shell allowed_commands in frontmatter are advisory — the actual enforcement is at the AI backend level

## Performance Impact
- Agent loading: single pass through `agents/*.md` — negligible
- Project discovery: file stat + read for `.kiro/` — <100ms typical
- Pipeline execution: dominated by AI backend response time, not moonbase logic

## Breaking Changes
- **Breaking:** `.json` agent files are no longer loaded. Must use `.md` format.
- **Breaking:** Old Agent struct fields (`Prompt`, `AllowedTools`, `ToolsSettings`, `Resources`, `WelcomeMessage`) replaced by new frontmatter-derived fields.
- **Non-breaking:** TUI key bindings unchanged, CLI subcommands unchanged.

## Dependencies on Other Work
- Agent `.md` files: ✅ already created
- Doctrine files: ✅ already exist
- Kiro CLI: external dependency, assumed installed

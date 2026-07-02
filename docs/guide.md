# How to Use Moonbase

A step-by-step guide from zero to running AI agents on your project.

---

## 1. Install

```bash
# From source (requires Go 1.24+)
git clone git@github.com:rk-senne/moonbase.git
cd moonbase
make build
cp bin/moonbase /usr/local/bin/

# Or from a release binary
# Download from GitHub Releases, extract, move to PATH
```

Verify it works:
```bash
moonbase help
moonbase status
```

---

## 2. Initialize Your Project

Navigate to any project and run:

```bash
cd ~/Projects/my-app
moonbase init
```

This creates:
```
.kiro/
├── specs/_templates/       ← spec templates (requirements, design, tasks)
├── steering/dev-rules.md   ← auto-detected project conventions
└── agents/                 ← 14 agent .md files ready for kiro-cli
```

The init command auto-detects your stack (Go, Java, Node, Python, Rust) and writes starter conventions.

---

## 3. Deploy Your First Agent

The simplest use case — ask one agent for help:

```bash
# Ask Numbuh 1 (Analyst) to break down a vague task
moonbase deploy 1 "we need user authentication"

# Ask Numbuh 4 (QA) to check recent changes
moonbase deploy 4 "verify the last commit didn't break anything"

# Ask Numbuh 274 (Security) to audit a file
moonbase deploy 274 "check the auth middleware for vulnerabilities"
```

**What happens:**
1. Moonbase loads the agent's full prompt (identity, rules, output format)
2. Discovers your project context (specs, steering rules, stack)
3. Composes everything together (steering + agent + specs + your task)
4. Hands off to kiro-cli for an interactive session

If kiro-cli isn't installed, the composed prompt is copied to your clipboard — paste it into Claude, ChatGPT, or any AI tool.

---

## 4. Run the Full Pipeline

For larger tasks, deploy the entire KND Council:

```bash
moonbase mission "add pagination to the /users API endpoint"
```

The pipeline runs 5 mandatory phases in sequence:

```
Phase 1: Numbuh 1 breaks your task into requirements (ACs)
Phase 2: Numbuh 2 designs the approach (files to touch, trade-offs)
Phase 3: Numbuh 3 implements (writes code, runs tests)
Phase 4: Numbuh 4 verifies (checks each AC, runs tests, classifies risk)
Phase 5: Numbuh 5 reviews (final check, prepares PR summary)
```

### Risk Gate

After Phase 4 (QA), the pipeline routes based on risk:

| Verdict | What Happens |
|---------|-------------|
| **LOW** | Proceed to review → done |
| **MEDIUM** | Back to Numbuh 3 for fixes (max 2 rework loops) |
| **HIGH** | Back to Numbuh 2 for redesign |
| **CRITICAL** | Pipeline STOPS — escalates to you |

### Conditional Specialists

After the core pipeline, specialists may trigger automatically:

- **Numbuh 0** fires if >5 files changed (architecture review)
- **Numbuh 274** fires if auth/secrets code was touched (security scan)
- **Numbuh 362** fires if CI/Docker/deploy configs changed (deployment check)

---

## 5. Understanding Agent Output

Every agent follows the same structure:

### Handoff Block
At the end of every response:
```
## Handoff

NEXT_AGENT: numbuh-4
REASON: Implementation complete, ready for QA
INPUT: 3 files changed, all tests passing
BLOCKERS: none
EVIDENCE: go test ./... exit 0
RISK: LOW
```

### Assumptions
When an agent isn't sure about something:
```
ASSUMPTION: Config loads at startup (based on existing pattern in main.go)
RISK_IF_WRONG: Runtime panic on missing config
REVERSIBLE: YES
```

### Questions
When an agent needs your input:
```
## Question for Human

QUESTION: Should pagination default to 20 or 50 items per page?
CONTEXT: No existing convention found in codebase
OPTIONS: A. 20 (standard REST), B. 50 (matches current frontend batch size)
DEFAULT: A (20) — industry standard
BLOCKING: NO — will proceed with 20 if no answer
```

---

## 6. When to Use Which Agent

| Situation | Agent | Command |
|-----------|-------|---------|
| "What should we build?" | Numbuh 1 (Analyst) | `moonbase deploy 1 "..."` |
| "How should we build it?" | Numbuh 2 (Architect) | `moonbase deploy 2 "..."` |
| "Build this specific thing" | Numbuh 3 (Implementer) | `moonbase deploy 3 "..."` |
| "Does this work correctly?" | Numbuh 4 (QA) | `moonbase deploy 4 "..."` |
| "Prep this for PR/review" | Numbuh 5 (Reviewer) | `moonbase deploy 5 "..."` |
| "Review the architecture" | Numbuh 0 (Overseer) | `moonbase deploy 0 "..."` |
| "Is this secure?" | Numbuh 274 (Security) | `moonbase deploy 274 "..."` |
| "Deploy this / fix CI" | Numbuh 362 (DevOps) | `moonbase deploy 362 "..."` |
| "Clean up dead code" | Numbuh 86 (Decommission) | `moonbase deploy 86 "..."` |
| "Write the docs" | Numbuh 999 (Documentation) | `moonbase deploy 999 "..."` |
| "Migrate/upgrade a library" | Numbuh 9 (Migration) | `moonbase deploy 9 "..."` |
| "What breaks with weird input?" | Numbuh 13 (Chaos) | `moonbase deploy 13 "..."` |
| "What is this old code?" | Sector Z (Legacy) | `moonbase deploy z "..."` |
| Full feature lifecycle | KND Council | `moonbase mission "..."` |

---

## 7. Customising Steering Rules

Edit `.kiro/steering/dev-rules.md` to teach agents your project's conventions:

```markdown
# Dev Rules

## Stack
- Language: TypeScript
- Framework: Next.js 14
- Test runner: vitest
- Build: `npm run build`
- Test: `npm test`

## Conventions
- Use server components by default, client only when needed
- API routes in app/api/ with Zod validation
- All database queries through Prisma
- Error handling: use Result<T> pattern, never throw
- CSS: Tailwind only, no custom CSS files

## Naming
- Files: kebab-case
- Components: PascalCase
- Functions: camelCase
- DB tables: snake_case
```

Agents read this automatically and follow your conventions instead of guessing.

---

## 8. Working with Specs

For non-trivial work, create a spec first:

```bash
# Copy the template
cp -r .kiro/specs/_templates .kiro/specs/user-auth

# Edit the files
vim .kiro/specs/user-auth/requirements.md
vim .kiro/specs/user-auth/design.md
vim .kiro/specs/user-auth/tasks.md
```

Or let Numbuh 1 create it for you:
```bash
moonbase deploy 1 "spec out a user authentication system with JWT and refresh tokens"
```

When specs exist, agents:
- Reference AC-IDs in their output (`Implements AC-1.3`)
- Verify against acceptance criteria (`AC-1.1: ✅ PASS`)
- Stay within documented scope
- Follow the design instead of inventing their own

---

## 9. Pipe Mode

For scripting and automation:

```bash
# Pipe a task to the full council
echo "add rate limiting to all API endpoints" | moonbase

# Pipe to a specific agent
echo "check for SQL injection" | moonbase deploy 274

# Use in scripts
git diff --name-only | moonbase deploy 4 "verify these changes"
```

---

## 10. Useful Commands

```bash
# Check your environment
moonbase status

# Validate your agents are properly formatted
moonbase lint

# View your config
moonbase config

# Install agents to a different project
cd ~/other-project
moonbase install --all

# See available operatives
moonbase list
```

---

## 11. Tips

### Start small
Deploy a single agent before running the full pipeline. Get comfortable with one operative first.

### Spec non-trivial work
If a task will touch >3 files or has ambiguity, spec it first. Agents produce dramatically better output when they have a spec to follow.

### Trust the risk gate
If Numbuh 4 says MEDIUM, let the rework loop happen. The second pass with specific QA feedback produces better code than forcing it through.

### Let agents question you
When an agent asks a question (UNCERTAIN level), answer it. A 10-second answer saves 10 minutes of wrong work.

### Read the handoffs
The handoff block tells you exactly what each agent did and what the next one needs. If something goes wrong, the handoff chain is your audit trail.

---

## Troubleshooting

### "Agent not found"
```bash
moonbase status    # check agents are loaded
moonbase lint      # check agent files are valid
```

### "No backend available"
Moonbase needs kiro-cli for interactive sessions. Without it, prompts are copied to clipboard.
```bash
which kiro-cli     # check if installed
```

### "No project context"
Agents work best with context. Run `moonbase init` in your project to create `.kiro/`.

### Pipeline gets stuck
Press `esc` twice to abort. Or wait — after 120 seconds a phase auto-times out.

---

*"Kids Next Door... ready for deployment."*

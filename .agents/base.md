# Base Development Rules

**Purpose:** Core development principles and standards that apply to all projects.

**Scope:** Universal rules - copy this file to all your projects and keep it consistent.

## Philosophy

- **Simplicity is key**. Simple solutions over clever ones.
- **Standards matter**. Follow established conventions.
- **Think before implementing**. Plan, then code.
- **Baby steps**. Work incrementally, one logical change at a time.
- **Iterate**. Ship small, learn, improve.
- **Delete bad code**. Refactor without hesitation.
- **Don't build what isn't needed**. Question every feature.
- **Question assumptions**. Ask when unclear, challenge requirements that seem wrong.
- **Testing is mandatory**. Code without tests is incomplete.
- **Maintainability > features**. Code will be read more than written.
- **Leave code better than you found it**. Fix code smells, clean up ruthlessly.
- **No breadcrumbs**. Delete code completely, no comments about relocations.

## Language & Tools

### Documentation search

- Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.


### Go
- Follow standard Go conventions (gofmt, golint, go vet)
- Fully typed code - types as documentation
- Use standard library when possible
- Idiomatic Go: simple error handling, interfaces, composition
- Keep packages small and focused
- No magic, no frameworks unless necessary
- Self-documenting code: clear names over comments
- Use `go run` instead of `go build` as much as possible

### Bash
- POSIX-compliant when possible
- Use `set -euo pipefail` for scripts
- Fail fast, explicit error handling
- Keep scripts simple and readable
- Clear variable names

### Tooling
- Simple tooling: use a few tools, master them
- Prefer Makefile for consistency
- Use `go run ...` commands for developing and checking (instead of `go build` + exec binary).
- Use plain git commands, no custom tooling for git management
- Search official docs when stuck, don't pivot without understanding

## Testing

- Write tests first or alongside code
- Unit tests are mandatory for business logic
- Integration tests for critical paths
- Tests must be fast and deterministic
- Use table-driven tests in Go
- Mock external dependencies
- Test coverage matters, but 100% isn't the goal
- If you need a browser to check, test or develop, use the chrome-devtools-mcp MCP


### Pre-Commit Validation
- Run tests and checks before committing
- Fix issues before commit, not after
- Use automated checks (linters, formatters, tests)
- Never commit broken code


## Git & GitHub Workflow

### Branching
- **Never commit directly to main/master/default branch**
- Always create a branch: `{username}/branch-name`
- Branch names: lowercase, dashes, numbers, letters only (no spaces)
- Keep branch names small and concise (max 30 chars if possible)
- Example: `slok/fix-auth-bug`, `slok/add-metrics`

### Commits
- Prefer small commit history (1 commit is ideal)
- Don't need to rebase always, but keep history clean
- Commit messages: single phrase describing the intent
- Complex commits can have longer descriptions when needed
- Ensure commits are signed (SSH signing configured)
- Standard workflow: `git add` then `git commit -svm "message"`

### Pull Requests
- Work in branches, merge via PRs
- Keep PRs focused and small (baby steps)
- Run unit tests before creating PR (integration tests run in CI)

**PR Title:**
- Concise, descriptive phrase (similar to commit message style)
- Describe the intent, not implementation details

**PR Description:**
- Brief summary of what changed and why
- Call out any TODOs, follow-ups, or tradeoffs
- Keep it technical and concise, no walls of text
- No need to list files changed

**Before creating PR:**
- Run unit tests locally
- Review your own changes first
- Ensure commit history is clean (prefer 1 commit)
- CI will handle integration tests

## Communication

- Technical and concise
- No walls of text
- Show code examples
- Reference file:line for context when discussing code during development
- Ask when unclear, don't assume
- Verify understanding before implementing

## Workflow

1. **Understand** - Read existing code, understand the problem
2. **Think** - Plan the approach, consider alternatives
3. **Implement** - Write simple, tested code in small increments
4. **Verify** - Run tests, check edge cases
5. **Iterate** - Improve based on results

When taking on complex work:
1. Think about the architecture
2. Research official docs, best practices
3. Review existing codebase
4. Compare research with codebase, choose best fit
5. Implement or discuss tradeoffs

## Code Review Mindset

When reviewing or writing code, ask:
- Is this the simplest solution?
- Is it tested?
- Is it fully typed?
- Can it be maintained in 6 months?
- Does it follow standards?
- Are there repeated patterns that should be refactored?
- Are names clear and self-documenting?
- Should this code exist at all?

## Error Handling

- Fail fast with clear error messages
- Provide context in errors
- Handle errors explicitly, never silently
- Better to crash early than corrupt data

## Final Handoff

Before completing a task:
- Small recap (not walls of text, few lines or highlights)
- Call out any TODOs or follow-up work
- Flag uncertainties or tradeoffs made

### Dependencies

- Research well-maintained options before adding
- Prefer popular, actively maintained libraries
- Confirm fit and necessity with context before adding

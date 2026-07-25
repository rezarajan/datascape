# Agentic development practices

**Class: Contract.**

platformctl's development process — most of it designed for and executed by Claude
sessions — worked demonstrably well even where the product didn't. These are the
process patterns worth re-adopting in datascape, with the known defects fixed. Most
become relevant only when code exists; they are recorded now so the repo is set up
correctly from the first line of code.

## Adopt from day one (docs-only phase included)

1. **Tiny CLAUDE.md (<200 lines), always.** Longer files measurably reduce how reliably
   agents follow them. CLAUDE.md carries only: the current phase, the one invariant of
   that phase, and pointers into the authoritative docs. Everything else goes in
   path-scoped rules or on-demand skills.
2. **Docs classified contract / plan / record**, declared on the docs map
   (`docs/README.md`), with records append-only.
3. **Compact instructions in CLAUDE.md**: what a context compaction must preserve
   (phase in progress, open questions, verbatim decisions) and what to discard
   (exploratory reading history).
4. **Resumability protocol**: any multi-session task keeps a `TASK_PROGRESS.md` at the
   working-tree root — step plan, per-step status, WIP commits — such that a session that
   dies mid-task is resumable by a different session from the file plus `git log` alone.
   Status lines are honest verdicts ("Docker leg COMPLETE and proven live; Kubernetes leg
   remaining"), and blocked items name *why* and *who* unblocks them.
5. **Deviations are findings, not judgment calls.** An agent that hits a mismatch between
   plan and reality stops at the smallest consistent state and reports, rather than
   silently adapting scope.
6. **Accept lists are run literally, not reasoned about.** "Would pass" is not a result.

## Adopt when code starts

7. **Enforcement hooks over instructions** for anything mechanical — hooks can't be talked
   out of compliance:
   - PostToolUse format-and-lint on edited files.
   - PreToolUse guard making plan/contract docs additive-only, with an auditable,
     gitignored marker-file unlock for human-authorized maintenance passes.
   - PreToolUse guard on subagent spawns denying silently-inherited expensive models.
   - **Fix from platformctl:** hook paths must be repo-relative (`$CLAUDE_PROJECT_DIR`),
     never absolute; platformctl's hooks broke on any other checkout.
   - **Fix from platformctl:** the additive-only guard needs a scheduled human
     consolidation pass baked into the process — pure accretion produced bloated,
     self-duplicating docs.
8. **Path-scoped rules** (`.claude/rules/`) that load only when matching files are
   touched: layering invariant, language style (with deliberate lint tunings documented
   so they aren't re-litigated), schema-change sync rules.
9. **Checked-in subagents** (`.claude/agents/`) pinning cheap models for high-volume,
   low-judgment work: test-runner that returns failures only, state-verifier that returns
   a diff vs expectation rather than raw dumps, read-only reviewers for contract
   compliance. Keep the roster small; every agent's prompt starts with the pre-coding
   checklist.
10. **Model economy as policy:** cheap models execute well-specified work; expensive
    models are reserved for genuine design judgment and root-cause-unknown investigation;
    verbose operations (test runs, log greps) are always delegated to context-absorbing
    subagents.
11. **Pre-coding checklist in CLAUDE.md** pointing at specific contract-doc sections
    (phase and exit criteria, final interface shapes, error-message contracts, acceptance
    scenario relevance, contract-suite existence, ADR coverage) — mirrored into subagent
    prompts so it happens automatically.
12. **Impact-mapped, content-hash-deduped integration testing** once an integration tier
    exists: a diff maps to affected suites; a shared ledger keyed on content-state hash
    prevents re-running suites already proven green against identical inputs, across
    branches and sessions; a meta-test fails if any integration test is unreachable from
    the impact map; suite runs serialize on a lock when they share a daemon.
13. **Operational hygiene rules learned from live incidents:** worktree inspection uses
    `git -C`, never a persisted `cd`; teardown never pattern-matches live infrastructure
    state (named objects only); no pipe-masked exit codes in commit/CI plumbing;
    credential/environment preflight before long test runs so suites fail on code, not
    on expired tokens.

## Known defects to not repeat

- Hardcoded absolute paths in `.claude/settings.json` hooks.
- The subagent-model guard duplicating the agent roster (drift risk) — derive from
  `.claude/agents/` frontmatter instead.
- Additive-only doc growth with no consolidation cadence.
- A `check` target that could not fail (`gofmt -l` exits 0 either way — compare output
  explicitly). Generalized: every gate must be demonstrated capable of failing.

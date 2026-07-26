# TASK_PROGRESS — solution setup + week-one plan

Resumability file per `docs/foundations/agentic-development.md` §4. Branch:
`claude/datascape-project-kickoff-ai83d9`. A session resuming this task needs only this
file plus `git log`.

## Step plan and status

1. **Golden-rules review (Q3.4)** — COMPLETE (commit `50c275f`). 67/70 rules bind;
   21/25/26 struck with reopen criteria; 8 compiler-shape interpretations. Recorded as
   the dated 2026-07-25 amendment in `docs/foundations/golden-rules.md` and under Q3.4
   in the problem definition. Q3.4 sign-off open item is closed.
2. **Repo setup per agentic-development.md** — COMPLETE (this commit). Hooks with
   repo-relative paths (`$CLAUDE_PROJECT_DIR`); classified-docs guard derives its
   protected set from `docs/README.md`; subagent-model guard derives its roster from
   `.claude/agents/` frontmatter (both platformctl drift defects fixed);
   doc-consolidation cadence (28d) with SessionStart reminder and
   `docs/consolidation.md` record; path-scoped rules (`.claude/rules/`); three
   checked-in subagents (test-runner, manifest-verifier: haiku; contract-reviewer:
   sonnet). Every gate demonstrated capable of failing AND passing (15/15 checks,
   run 2026-07-25).
3. **Commit-style enforcement (owner directive, 2026-07-25)** — COMPLETE. Strict
   google/conventional style gated by `.githooks/commit-msg` (activated via
   `core.hooksPath` by a SessionStart hook); gate demonstrated failing and passing
   (10/10 checks). The two unpushed commits were reworded to conform. The four
   pre-existing commits (`7c38429..c24f4ec`, shared with `origin/main`) keep their
   messages — rewriting shared history is an owner decision, not taken.
4. **Week-one plan** — WRITTEN at `docs/plans/01-week-one.md`; status: **APPROVED
   2026-07-26** (Revision A). Istio ambient confirmed; managed-emit artifact deferred
   to week two.
5. **Problem-definition re-sign-off** — COMPLETE (commit `9b8f716`, 2026-07-26).
   Amendment 1 (guarantees-compiler) and Amendment 2 (trust-boundary) both
   re-signed-off in one owner confirmation; also fixed an inconsistent prior partial
   flip (commit `1248528` had updated only the top status line, not the amendment
   footers or `docs/README.md`).
6. **Week-one build** — IN PROGRESS. Build order per Revision A: (a) Go module +
   hexagonal scaffold + arch test, (b) Stack+postgres declaration schema
   (guarantees + placement), (c) compiler core + Flux emitter with golden-file/
   determinism tests, (d) zero-trust slice, (e) durability slice, (f) acceptance
   harness on kind. Status of each sub-step tracked here as it lands.

## Open items owned by the owner

- **Q1.1a**: team name + developer-customer count — record (privately if repo goes
  public) before the first dogfood week, so adoption claims have a denominator. Still
  open as of 2026-07-26.
- **Managed-emit artifact** (deferred at Revision A approval, blocks week two only):
  Flux tf-controller `Terraform` CR (recommended) vs plain Terraform module vs
  Crossplane claim.

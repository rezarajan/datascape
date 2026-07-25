---
globs:
  - "cmd/**"
  - "internal/**"
  - "pkg/**"
  - "go.mod"
---

# Source rules (loaded when touching product code)

Run the pre-coding checklist in CLAUDE.md before writing anything.

- **Layering (golden rule 8, hexagonal):** domain imports nothing; ports import only
  domain; adapters implement ports; only the composition root knows concrete adapters.
  An architecture test in CI enforces this mechanically — if you add the first adapter,
  add the arch test in the same commit.
- **Named planes (rule 9):** declaration model / compiler core / target emitters /
  verification are the named layers. Cross-component wiring logic lives in the compiler
  core — never precipitated into a component adapter.
- **No mutating verbs against any backend.** Datascape compiles; Flux applies (Q3.2).
  Adding any write-intent cluster call trips the reopen criteria of struck rules
  21/25/26 (see the 2026-07-25 amendment in `docs/foundations/golden-rules.md`) and is
  a scope finding, not a judgment call.
- **Determinism is a tested feature (rules 22, 45):** same declaration → byte-identical
  compiled output, golden-file tested. Machine output is a byte-stable contract.
- **Fail closed (rules 34, 37, 50):** unimplemented paths refuse loudly ("planned, not
  yet available"); security properties have no best-effort tier; a schema-accepted
  field nothing consumes is a defect.
- **Errors carry the remedy (rule 35).** No fixed-duration sleeps anywhere (rule 44).

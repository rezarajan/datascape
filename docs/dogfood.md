# Dogfood record

**Class: Record (append-only).** Dated notes from real use of d7s, each measured
against the Q2.1 target (declaration to running, verified stack in under one hour)
and feeding the kill review (Q2.2: 4 weeks / 2+ real stacks from the first dogfood
week).

## Dogfood week one — opened 2026-07-26

**Clock:** the first dogfood week starts today, so the kill review falls due
**2026-08-23**, requiring 2+ real stacks by then. (Noted, dormant contradiction: the
week-one plan says the clock "starts when this week [the build week] begins" while
CLAUDE.md and the steward protocol say "from the first dogfood week" — both readings
give the same date because the artifact was built and dogfooding started on the same
day, 2026-07-26. If a future reader needs the clock's authority, this is flagged
here rather than reconciled.)

**Setup decisions (owner, 2026-07-26, steward option rounds — selections verbatim):**

- First stack: **"A real request I have"**, then, asked for the shape:
  **"Internal service/API database"**. The specifics (service name, consumer
  identity, database purpose) were requested in both rounds and not provided; the
  operator run below uses operator-chosen placeholder names, and renaming later is a
  recompile — itself dogfood data. The realness of the demand rests on the owner's
  attestation.
- Substrate: **"kind, locally"** — chosen over a real cluster knowing the evidence
  is weaker (no real workload depends on a local kind cluster); recorded as such for
  the kill review.
- Driver: **"I drive as operator"** — the steward executed the flow. The kill review
  should read every timing below as **agent-assisted operator** evidence: no human
  hand-typed the commands.
- Q1.1a denominator: **5 developers** (recorded in the problem definition,
  2026-07-26; team name still open).

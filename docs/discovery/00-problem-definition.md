# Problem definition

**Class: Plan → Contract.**
**Status: OPEN — awaiting answers.**

This document becomes the contract that authorizes solution work. It reaches
**Signed off** only when every round below has recorded answers and the owner has
explicitly marked it so. Until then, no product code.

The questions are ordered so that later rounds depend on earlier ones. Answers are
recorded inline under each question, dated. Where an answer invalidates a later
question, the question is struck with a reason, not deleted.

platformctl's failure mode is the template for what this document prevents: it answered
Round 3 (solution shape) brilliantly without ever answering Rounds 1–2 (problem and
evidence). See `../foundations/lessons-from-platformctl.md`.

---

## Round 1 — The problem and the person

**Q1.1 — Who is the ONE primary user of datascape v1?**
Not a persona list — one person-shaped answer, ideally nameable people. platformctl
carried three personas whose needs pulled in different directions, and the vaguest one
justified the heaviest architecture. Candidates from the platformctl record:
(a) a data/platform engineer standing up local dev environments;
(b) CI pipelines needing reproducible ephemeral data stacks;
(c) a platform team building an internal developer platform;
(d) you, Reza, operating a real platform of your own.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Platform team / IDP** — a team
> building an internal developer platform on top of datascape's model.
>
> *Finding (flagged, not reconciled):* this is the persona the platformctl post-mortem
> identified as the one that "quietly justified most of the architectural weight without
> any named consumer." Selecting it deliberately is legitimate, but it raises Q1.1a below,
> which must be answered before this round closes.

**Q1.1a — Name the platform team.** Which concrete team (even if it's a team of one, or a
future team you intend to be) is the customer? Who are *its* customers — how many
developers would consume the platform it builds with datascape?

> *Answer:*

**Q1.2 — What is the ONE job-to-be-done that datascape must nail?**
Stated as a job, not a feature: "when I ___, I want to ___, so I can ___." What is the
painful, recurring task this replaces?

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Stand up a data stack fast** —
> from empty machine to a working broker/database/storage/wiring stack in minutes,
> declaratively.
>
> *Synthesis note:* combined with Q1.1, the job statement becomes second-order: the
> platform team's own job is *giving their developers* fast, self-service data stacks.
> The v1 job should therefore be phrased from the platform team's seat — e.g. "when a
> product team asks for a data stack, I want to hand them a declared, reproducible one in
> minutes, so I stop being the bottleneck." Q1.2a below pins this down.

**Q1.2a — Whose hands are on the tool?** Does the platform team run datascape and hand
developers endpoints, or do developers self-serve from templates the platform team
publishes? The two imply very different v1 surfaces (operator CLI vs golden-path
templates).

> *Answer:*

**Q1.3 — What do people doing this job use today, and where exactly does it hurt?**
The honest alternatives: Compose files + init scripts, Testcontainers, Tilt/Skaffold,
Helm charts + operators, Terraform + scripts, managed cloud services, or "a wiki page
and patience." For the chosen user: which one do they actually use, and what is the
specific, observed failure — not the inferred one?

> *Answer:*

**Q1.4 — What evidence exists (or will be gathered) that this pain is real and felt by
more than one person?**
Interviews, your own recurring experience, community signals, failed searches for an
existing tool. If the answer is "none yet," Round 1 isn't done — the evidence-gathering
plan is the answer.

> *Answer:*

**Q1.5 — Why did platformctl-the-product not solve this?**
Separate from why it failed as a project. If the defined problem is one platformctl's
scope already covered, what specifically about its shape was wrong for the job — too many
concepts? wrong workflow? wrong runtime target? This determines how much of its design
thinking (not code) is reusable.

> *Answer:*

## Round 2 — Success and failure

**Q2.1 — What is the outcome-facing success criterion for v1?**
Mechanism criteria (idempotency, drift detection) belong to engineering. This one must be
observable from outside: e.g. "user X goes from empty machine to working pipeline in
N minutes and uses it again the next week without being asked."

> *Answer:*

**Q2.2 — What would make you kill the project, and by when?**
The pre-registered failure condition. platformctl never had one, so it accreted plans
instead of concluding. A kill criterion is the cheapest form of intellectual honesty.

> *Answer:*

**Q2.3 — What is datascape explicitly NOT, this time?**
platformctl's non-goals list existed but didn't hold (lakehouse catalogs, query engines,
zero-trust networking all arrived anyway). Name the three most tempting adjacent scopes
and pre-commit to refusing them for v1.

> *Answer:*

## Round 3 — Solution shape (locked until Rounds 1–2 are answered)

**Q3.1 — What is the smallest complete slice?**
One worked scenario — the walking skeleton — that is simultaneously the docs example, the
acceptance test, and the demo, exercised exactly the way a user would invoke it, from
week one.

> *Answer:*

**Q3.2 — Build on, wrap, or build new?**
Given Round 1's answer: does datascape orchestrate existing tools (Compose, Helm,
Terraform), extend one, or own the reconciliation loop itself the way platformctl did?
The platformctl record proves owning the loop is expensive; the job may not need it.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Decide after Rounds 1–2** —
> the solution shape is locked only after problem and evidence are answered, per this
> document's ordering.

**Q3.3 — What is the v1 runtime/deployment surface?**
One substrate, chosen by the user's actual environment — not an abstraction over several.
(Rule 11 still applies when a second one arrives: the second implementation is the test
of the seam. It just doesn't have to arrive in v1.)

> *Answer:*

**Q3.4 — Which golden rules bind v1?**
A one-time review of `../foundations/golden-rules.md` against the defined problem; rules
that don't apply are struck with a recorded reason.

> *Answer:*

## Round 4 — Validation plan

**Q4.1 — How is the problem definition validated before heavy building?**
The cheapest artifact that tests the riskiest assumption: a README-driven design shared
for feedback, a prototype in a week, a manual walkthrough of the job with the existing
tools while recording friction. Name the artifact, the audience, and the date.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Dogfood week one** — define
> the walking-skeleton scenario, build the smallest thing that runs it, and use it on a
> real workload immediately; an agent and/or the owner attempting real use continuously,
> not at the end.

**Q4.2 — Who dogfoods, on what real workload, and how often?**
An agent attempting real use was what exposed platformctl — that feedback loop arrives
in week one this time, not at the end.

> *Answer:*

---

## Sign-off

- [ ] Rounds 1–4 answered and dated
- [ ] Owner review: the answers are consistent with each other
- [ ] Status above flipped to **SIGNED OFF**, with date

Once signed off, this document is a contract: scope changes reopen it explicitly
(a dated amendment naming what changed and why), and `golden-rules.md` rule 4 applies —
every scope growth re-answers "who asked?"

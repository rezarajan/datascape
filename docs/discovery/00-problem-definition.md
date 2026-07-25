# Problem definition

**Class: Contract (promoted from Plan at sign-off).**
**Status: SIGNED OFF — 2026-07-25, by the owner, via kickoff questionnaire.**

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

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **A real team at my work** — a
> named, existing platform team the owner is on, with real developers as customers.
>
> *Open item (non-blocking, noted at sign-off):* the team's actual name and its
> developer-customer count are not written down here. Record them (privately if the repo
> is public) before the first dogfood week, so adoption claims have a denominator.

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

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Operator first, self-serve
> later** — v1 is operated by the platform team, which hands developers working
> endpoints/credentials; the developer self-serve surface is an explicit later phase with
> its own discovery. v1 surface = operator CLI + declarations.

**Q1.3 — What do people doing this job use today, and where exactly does it hurt?**
The honest alternatives: Compose files + init scripts, Testcontainers, Tilt/Skaffold,
Helm charts + operators, Terraform + scripts, managed cloud services, or "a wiki page
and patience." For the chosen user: which one do they actually use, and what is the
specific, observed failure — not the inferred one?

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Managed cloud services** —
> MSK/RDS/S3-class services wired together by console and Terraform. The pain is the
> wiring and reproducibility, not the hosting.
>
> *Strategic frame (owner, same day, follow-up):* the incumbent is also what the team is
> leaving. Datascape is part of a **deliberate move off managed services onto self-hosted
> Kubernetes** — standing stacks up fast, with a security posture that justifies
> self-hosting, is what makes that migration viable. (Runtime consequence recorded under
> Q3.3.)

**Q1.4 — What evidence exists (or will be gathered) that this pain is real and felt by
more than one person?**
Interviews, your own recurring experience, community signals, failed searches for an
existing tool. If the answer is "none yet," Round 1 isn't done — the evidence-gathering
plan is the answer.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **"I'm on the team, feel it
> weekly"** — first-hand, recurring pain: the owner does this wiring themselves and it
> costs real hours regularly. Strongest evidence short of a paying customer; the 4-week
> dogfood window (Q2.2) is the check that proximity hasn't inflated it.

**Q1.5 — Why did platformctl-the-product not solve this?**
Separate from why it failed as a project. If the defined problem is one platformctl's
scope already covered, what specifically about its shape was wrong for the job — too many
concepts? wrong workflow? wrong runtime target? This determines how much of its design
thinking (not code) is reusable.

> *Answer (2026-07-25, derived by synthesis from the owner's other answers — not asked
> directly; owner accepted by signing off the document containing it):* three shape
> errors, per the answers above and below. (a) **Wrong runtime target**: single-host
> Docker, while this user's destination is Kubernetes. (b) **Wrong posture**: platformctl
> owned the reconciliation loop; a GitOps-native platform team needs a compiler feeding
> the cluster's existing reconciler (Q3.2). (c) **The differentiator was an accretion**:
> zero-trust arrived late and never soaked, when for this problem it is the pitch itself
> (Q3.1). Its design thinking about resource modeling and cross-component wiring remains
> relevant; its runtime, engine, and layering machinery do not.

## Round 2 — Success and failure

**Q2.1 — What is the outcome-facing success criterion for v1?**
Mechanism criteria (idempotency, drift detection) belong to engineering. This one must be
observable from outside: e.g. "user X goes from empty machine to working pipeline in
N minutes and uses it again the next week without being asked."

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Time-to-stack collapses.**
> Today a new stack costs days-to-weeks of wiring and review; the v1 target is **under
> one hour** from declaration to working endpoints with policies enforced, measured on
> real requests — speed is the metric even if team-wide adoption is partial at first.

**Q2.2 — What would make you kill the project, and by when?**
The pre-registered failure condition. platformctl never had one, so it accreted plans
instead of concluding. A kill criterion is the cheapest form of intellectual honesty.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Dogfood loses to the status
> quo — 4 weeks, 2+ real stacks.** After four weeks of continuous dogfooding, with at
> least two real stack stand-ups attempted, if standing up a real stack is still faster
> or safer the old way: kill or pivot. The clock starts at the first dogfood week (Q4.2).

**Q2.3 — What is datascape explicitly NOT, this time?**
platformctl's non-goals list existed but didn't hold (lakehouse catalogs, query engines,
zero-trust networking all arrived anyway). Name the three most tempting adjacent scopes
and pre-commit to refusing them for v1.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* refused for v1:
> **multi-runtime abstraction** (one Kubernetes target; the Flux seam stays thin — see
> Q3.3) and **day-2 operations** (backup/restore, HA tuning, upgrades, continuous drift
> *repair* — day-0 stand-up is the whole v1 job).
>
> *Finding (flagged, not reconciled):* the question asked for three refusals; the owner
> gave two, and deliberately pulled the other candidates INTO scope — zero-trust
> networking is the differentiator (Q3.1) and the lakehouse is the acceptance workload
> (Q4.2). This is the scope profile whose weight the platformctl record warns about. The
> mitigations, chosen consciously: both live in the walking skeleton from week one
> (soaked, not late-stage accretions), the week-one artifact is deliberately small
> (Q4.2), and the 4-week kill criterion (Q2.2) is the backstop.

## Round 3 — Solution shape (locked until Rounds 1–2 are answered)

**Q3.1 — What is the smallest complete slice?**
One worked scenario — the walking skeleton — that is simultaneously the docs example, the
acceptance test, and the demo, exercised exactly the way a user would invoke it, from
week one.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* the walking skeleton is a
> declared data stack of **Kafka-class broker, Postgres-class database, object storage,
> and CDC/connect wiring, with service mesh + zero-trust across all components and
> verifiable compute**, compiled to Flux-consumable manifests; the real workload it must
> serve is a **lakehouse deployment** (Q4.2).
>
> **Zero-trust's role (owner):** *it IS the differentiator* — the pitch is "zero-trust
> data platform by default"; the reason to leave managed services is getting a security
> posture the console can't give. A skeleton without it proves nothing.
>
> **"Verifiable compute," owner definition (three selections):**
> 1. *Supply-chain attestation* — signed images, SBOM/provenance, admission refuses
>    unsigned/unattested workloads (verify WHAT is running);
> 2. *Policy admission proofs* — every deployed object provably passed policy, with an
>    auditable decision log (verify the rules held);
> 3. *Declared = running* — continuous verification that cluster state matches the
>    signed, compiled declaration (verify the platform is what the manifest says).
>
> TEE/confidential compute was offered and **not** selected for v1.
>
> The skeleton is the v1 scope, not the week-one artifact — week one builds one thin
> slice of it (Q4.2).

**Q3.2 — Build on, wrap, or build new?**
Given Round 1's answer: does datascape orchestrate existing tools (Compose, Helm,
Terraform), extend one, or own the reconciliation loop itself the way platformctl did?
The platformctl record proves owning the loop is expensive; the job may not need it.

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Decide after Rounds 1–2** —
> the solution shape is locked only after problem and evidence are answered, per this
> document's ordering.
>
> *Decided later the same day, after Rounds 1–2 closed:* **GitOps compiler.** Datascape
> compiles a stack declaration into Kubernetes manifests for the cluster's GitOps
> machinery to apply. It owns **no runtime reconciliation loop of its own** — the GitOps
> engine is the reconciler. This is the sharpest break from platformctl, whose record
> prices owning the loop.

**Q3.3 — What is the v1 runtime/deployment surface?**
One substrate, chosen by the user's actual environment — not an abstraction over several.
(Rule 11 still applies when a second one arrives: the second implementation is the test
of the seam. It just doesn't have to arrive in v1.)

> *Answer (2026-07-25, owner, via kickoff questionnaire):* **Kubernetes**, singular —
> chosen because the team is deliberately migrating off managed cloud services onto
> self-hosted k8s (Q1.3). Verbatim on the delivery plane: *"v1 should target Flux, but
> the actual target should be hidden behind an interface."*
>
> *Flag (rule-11 tension, noted not reconciled):* an interface with one implementation is
> exactly the seam the golden rules say gets proven only when the second consumer
> genuinely arrives. Resolution consistent with both: keep the interface as thin as the
> compiler's output boundary (emit-manifests-for-target), design nothing for a
> hypothetical Argo/other backend until one is actually needed.

**Q3.4 — Which golden rules bind v1?**
A one-time review of `../foundations/golden-rules.md` against the defined problem; rules
that don't apply are struck with a recorded reason.

> *Answer (2026-07-25):* not yet performed. Recorded as the **first solution-phase task**,
> to run alongside repo setup per `../foundations/agentic-development.md`, before any
> product code. The owner signed off with this item explicitly open.

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

> *Answer (2026-07-25, owner, via kickoff questionnaire):*
> **Who:** the owner (on the platform team, feels the pain weekly — Q1.4) plus an agent
> attempting real use, continuously from week one.
> **Workload:** a real **lakehouse deployment** — the broker, CDC, and object storage
> exist to feed it, so it is the acceptance scenario, not an add-on.
> **Week-one artifact:** *compiler + one component + mTLS* — a declaration compiled to
> Flux manifests for ONE component (e.g. a Postgres-class database via a
> CloudNativePG-class operator), deployed with mesh mTLS on. The smallest honest slice
> that proves compile-to-*secure*-running; the remaining components, the verifiability
> guarantees, and the lakehouse workload build out from there toward the full skeleton.
> **Cadence:** continuous; kill review at 4 weeks / 2+ real stacks (Q2.2).

---

## Problem statement (synthesis, 2026-07-25)

A real platform team (the owner's team) is deliberately migrating its data
infrastructure off managed cloud services onto self-hosted Kubernetes. Today, standing
up a new data stack costs days to weeks of console-and-Terraform wiring, and the
security posture that would justify self-hosting does not exist by default. **Datascape
is a GitOps compiler**: the platform team declares a data stack once, and datascape
compiles it into Flux-consumable Kubernetes manifests in which zero-trust (mesh
identity, mTLS, policy) and verifiability (supply-chain attestation, policy admission
proofs, declared = running) hold by default. It owns no reconciliation loop — the GitOps
engine is the reconciler.

**Success:** request → running, policy-enforced stack in under one hour, on real
requests. **Kill:** after 4 weeks of continuous dogfooding and 2+ real stack stand-ups,
the old way is still faster or safer. **v1 refuses:** multi-runtime abstraction and
day-2 operations. **v1 embraces, consciously:** zero-trust as the differentiator and a
lakehouse deployment as the acceptance workload — the scope profile platformctl's record
warns about, taken with eyes open and a kill switch armed.

Findings flagged during discovery (recorded inline above, not reconciled): the unnamed
platform team (Q1.1a), the two-of-three refusals with the other candidates pulled into
scope (Q2.3), and the rule-11 seam tension on the Flux interface (Q3.3).

## Sign-off

- [x] Rounds 1–4 answered and dated — with two items explicitly open at sign-off:
      the golden-rules review (Q3.4, first solution-phase task) and the team's recorded
      name/denominator (Q1.1a, before first dogfood week)
- [x] Owner review: conducted via the kickoff questionnaire, 2026-07-25; inconsistencies
      recorded as flagged findings inline rather than reconciled silently
- [x] Status above flipped to **SIGNED OFF**, 2026-07-25 — mechanism: owner selected
      *"Sign off now via this answer"* in the kickoff questionnaire (Q4.3)

Once signed off, this document is a contract: scope changes reopen it explicitly
(a dated amendment naming what changed and why), and `golden-rules.md` rule 4 applies —
every scope growth re-answers "who asked?"

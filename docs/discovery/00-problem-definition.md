# Problem definition

**Class: Contract (promoted from Plan at sign-off).**
**Status: SIGNED OFF — 2026-07-25, by the owner, via kickoff questionnaire.
Amended same day: Amendment 1 (guarantees-compiler reframing) and Amendment 2
(trust-boundary model) — see end of document.
Re-sign-off: SIGNED OFF — 2026-07-26, by the owner, covering both amendments (see
"Re-sign-off" section at the end of this document).**

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
>
> *Answer (2026-07-26, owner, via steward decision round at dogfood start):* selected
> **"Small team: 2–5 devs"**, then, asked for the exact figure, **"5 developers"** —
> the adoption denominator is **5**. The team's actual name was requested in both
> rounds and not provided; it remains the open sliver of this item, recorded here
> rather than blocking the dogfood start the owner directed. *(Finding, not
> reconciled: the sign-off note above asks for name AND count before the first
> dogfood week — the count is now recorded, the name is not.)*

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
> *(Amended by Amendment 1, 2026-07-25: all-or-nothing migration replaced by hybrid
> placement — managed where warranted, k8s where suitable, chosen per component against
> its declared guarantees.)*

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
> *(Refined by Amendment 1, 2026-07-25: hybrid placement enters scope via exactly one
> seam-proving component pair; arbitrary-substrate abstraction stays refused.
> PROVISIONING recovery machinery is day-0 and in scope; OPERATING it stays day-2 and
> refused.)*
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
> *(Superseded by Amendment 1, 2026-07-25: the differentiator is the guarantees
> compiler; zero-trust becomes its flagship guarantee family.)*
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
>
> *Answer (2026-07-25, review performed as the first solution-phase task):* **Review
> complete — 67 of 70 rules bind; rules 21, 25, and 26 are struck.** All three fall for
> the same structural reason: they constrain a mutating runtime engine and an owned
> state store, and the signed-off shape (GitOps compiler, no reconcile loop, no mutating
> verbs, git as the only state) has neither. Each struck rule keeps its text and carries
> a reopen criterion — any future mutating adapter or persisted state reinstates it.
> Eight rules (11, 20, 23, 24, 27, 29, 30, 63) carry recorded compiler-shape
> interpretations with text unchanged. Full outcome: the dated 2026-07-25 amendment at
> the end of `../foundations/golden-rules.md`. With this, every un-struck rule binds,
> and this Q3.4 open item from sign-off is **closed**.

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

*(Superseded in part by Amendment 1, same day — the amended statement at the end of this
document is authoritative. This text is kept per the no-deletion rule.)*

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

---

## Amendment 1 — 2026-07-25: the guarantees-compiler reframing

**What changed and why:** the owner, reviewing the signed-off definition end-to-end the
same day, found it "does not capture the intent" and reopened it. Recorded here per the
docs rules: statement verbatim first, contradictions flagged, re-answers dated.

### Owner statement (verbatim, 2026-07-25)

> I've gone ahead and updated the main branch on the datascape repo. However, I do feel
> like it does not capture the intent. As such, I would like to re-evaluate the product
> (with you being the practicality anchor). You see, many of the actually useful
> products offer a convenience function to getting high-quality, production-grade
> results. For example, dbt delivers on the premise of 'Build data pipelines the way
> software engineers build apps: modular, tested, and version-controlled. Native SQL
> comprehension and local validation catch issues before they hit the warehouse, while
> multi-dialect compilation keeps logic portable as your data platform evolves" - a tool
> that truly solves many latent issues upfront. Terraform helps solve infrastructure
> provisioning headaches by enabling infrastructure as code across many providers -
> enabling engineers to rapidly provision production-grade infra they can trust.
> Kubernetes also extends this IaC model - and especially Talos Linux, which helps to
> realiably provision production-grade Kubernetes and make its management super
> straightforward - a win all-around because convenience here actually means you ship
> faster and more reliabliy. For me, there is this gap in the data platform space - if
> you're not using a cloud data platform, you're out-of-luck in terms of convenience.
> Even then, migrating from one provider to the next is always a headache because there
> is always tight coupling. Now - the thing about data is that you really do require
> reliable infrastructure, and maybe Kubernetes alone cannot provide that (at least
> without much headache involved and the actual requirement for super heavy-duty worker
> notes for production databases); there is tangible benefit in using cloud-provider
> managed infra like databases or object storage, but at the same time some applications
> do not always need managed services, and can indeed run on a K8S cluster (which is
> portable, for the most part). If you noticed, across all the example tools I
> mentioned, they can be run at production-grade levels across any scale - a small
> startup, a hobbyist, a mid-sized company, a one-off project, or even to super
> large-scale complex enterprises. They all do this by realizing a SYSTEM that functions
> deterministically across all these levels; the system itself does the heavy-lifting of
> ensuring everything works; and at-scale deterministic functionality (formal proofs for
> example) guarantee things will always work. To tie it all in - d7s is supposed to
> remediate that gap I identified for data engineers themselves. It gets data engineers
> to stop thinking about implementation and rather the declaration of their platform's
> system, knowing that d7s will make sure all the guards are in-place. I need you to
> help me refine this idea, actually identify the problem statement, and develop a
> project around it.

### Findings — contradictions with the signed-off text (flagged, then resolved below)

1. **Primary user** (Q1.1): platform team / IDP → statement says "data engineers
   themselves," at any scale. *Resolved by A3: beachhead unchanged (the owner's team);
   any-scale is vision, not v1 contract.*
2. **Placement** (Q1.3 strategic frame): "deliberate move off managed" →
   managed-where-warranted hybrid. *Resolved by A2: placement becomes a declared,
   compiler-validated binding.*
3. **Multi-runtime refusal** (Q2.3): placement portability is now the point. *Resolved
   by A2: one seam-proving pair in v1; arbitrary-substrate abstraction stays refused.*
4. **Differentiator** (Q3.1): "zero-trust IS the pitch" → superseded. *Resolved by A1:
   the guarantees compiler is the pitch; zero-trust is its flagship guarantee family.*

**Practicality-anchor corrections accepted by the owner** (via the A1–A4 round): no
"formal proofs" claim — the honest v1 promise is determinism + fail-closed compilation +
conformance-tested substrates; no all-scales claim in the v1 contract — the exemplars
(dbt, Terraform, Kubernetes, Talos) each launched on one small load-bearing primitive
and one beachhead, and earned generality later.

### Re-answers (2026-07-25, owner, via re-evaluation questionnaire)

- **A1 — Differentiator: the guarantees compiler.** The product is the
  guarantee-checked platform graph — durability, recovery, security, and wiring
  correctness declared per component and enforced at compile time. Zero-trust is the
  flagship guarantee family, not the product identity.
- **A2 — v1 substrates: one seam-proving pair.** v1 compiles exactly ONE component both
  ways — Postgres as CNPG-on-Kubernetes AND as a managed cloud database — and no other
  component gets a managed variant. d7s still never applies anything: the managed
  placement compiles to declarative artifacts applied by existing machinery. *Open
  design question (decide at week-one plan revision):* the managed-emit artifact — Flux
  tf-controller `Terraform` CR, plain Terraform module, or Crossplane claim. The first
  two keep one delivery plane (Flux); all three preserve the no-mutating-verbs posture.
- **A3 — Beachhead: still the owner's platform team.** The evidence (first-hand weekly
  pain, Q1.4) is unchanged; only the product thesis widened. The any-scale, "data
  engineers everywhere" framing is the vision statement, carried in the README, not a
  v1 commitment.
- **A4 — Survivors, unchanged:** GitOps-compiler posture (no owned reconcile loop, no
  mutating verbs); the kill criterion (4 weeks / 2+ real stacks — the clock restarts at
  the amended contract's first dogfood week); week-one dogfood validation (revised to a
  guarantee-bearing slice); the lakehouse acceptance workload.

### The guarantee primitive (new, binding)

Every declared guarantee ships as a **triple**: (1) a compile-time check, (2) emitted
infrastructure that provides it, (3) a conformance probe that proves it against the
running stack. A guarantee that cannot do all three does not ship — fail closed, no
best-effort tier (golden rules 34/37/50). v1 guarantee families: **transport security**
(mesh mTLS + compiled least-privilege authorization), **durability/recovery**
(scheduled backups / RPO declared per component — provisioning is day-0 and in scope;
operating restores stays day-2 and refused), **wiring correctness** (cross-component
bindings validated at compile time). Attestation/admission-proof/declared=running
families remain skeleton scope as before (Q3.1).

### Amended problem statement (authoritative; supersedes the synthesis above)

Data engineers assembling platforms outside a single cloud vendor must hand-build
production-grade guarantees — durability, recovery, security, wiring correctness — from
infrastructure primitives that don't understand data, and every managed-vs-self-hosted
choice hard-couples the architecture to a provider. **d7s is a compiler for data
platforms**: declare the system — components, data flows, and the guarantees each must
meet — and d7s compiles it, deterministically, to substrates that can honor those
guarantees, refusing to compile a platform that can't. Placement (managed service vs
operator-on-k8s) is a declared binding the compiler validates, not an architecture
rewrite. d7s owns no reconcile loop and mutates nothing: the GitOps engine applies what
d7s compiles.

**Beachhead:** the owner's platform team, on real stack requests. **Success:** request →
running, guarantee-enforced stack in under one hour. **Kill:** 4 weeks / 2+ real stacks,
dogfood loses to status quo. **v1 refuses:** arbitrary-substrate abstraction (one seam
pair only), day-2 operation, TEE/confidential compute, developer self-serve.

### Consequences

- `docs/plans/01-week-one.md` requires revision before owner approval (revision proposed
  there, dated).
- The golden-rules Q3.4 amendment is unaffected: no rule's binding status changes; the
  struck rules' reopen criteria (21/25/26) still hold since d7s still never mutates.
- Open items carried forward: team name/denominator (Q1.1a), managed-emit artifact
  choice (A2), and re-sign-off of this amendment by the owner.

**Amendment status: recorded 2026-07-25; RE-SIGNED-OFF 2026-07-26 (see "Re-sign-off"
section at the end of this document).**

---

## Amendment 2 — 2026-07-25: the trust-boundary model

**What changed and why:** the owner proposed a simplifying assumption — d7s trusts only
its own managed infra, everything outside is untrusted and user-piped — and asked for it
to be corrected and refined against how the exemplar tools (dbt especially) handle
safeguard guarantees. Recorded per docs rules: statement verbatim, correction, dated
decisions.

### Owner statement (verbatim, 2026-07-25)

> Now, let's make some simplifying assumptions: all the example solutions I mentioned to
> you generally work on the premise that they are fully-managed by the actual tool
> themselves (correct me if I am wrong, but I think this is the design intent).
> Similarly, d7s may adopt a decision that it can only control and trust its own managed
> infra - anything outside of that will be not be trusted, and it will be entirely up to
> the user to pipe those resources into the platform itself, or from platform to an
> outside resource. Help me correct and refine this idea - I am particularly interested
> in how dbt handles these strong safeguard guarantees.

### Correction accepted by the owner (via the B1–B4 round)

The exemplars are not walled gardens; they are **provenance-trust systems with declared
gates**. Terraform trusts only its own state, never mutates what it doesn't own, and
represents the outside as read-only data sources plus an explicit import ceremony.
Kubernetes controllers reconcile only owned objects. dbt's `ref()`/`source()` dichotomy
is the pattern d7s adopts: everything dbt builds carries total guarantees (order,
lineage, tests) *because it built it*; everything it didn't build must be declared as a
named `source()` — never written to, but checked at the gate (freshness thresholds,
schema tests) — and guarantees attenuate **explicitly, never silently** past that gate.
"Outside is entirely the user's problem" is rejected: it recreates undeclared spaghetti
exactly where guarantees end — the pre-dbt world at the platform's edge.

### Decisions (2026-07-25, owner, via refinement questionnaire)

- **B1 — Trust boundary: provenance.** Inside = anything d7s compiled/provisioned —
  including the seam pair's managed database. Outside = anything it didn't. Terraform
  state / dbt-ref semantics; the hybrid story stays first-class.
- **B2 — Egress: compiled default-deny.** The allowlist (mesh ServiceEntry +
  authorization) is generated only from declared externals and declared wiring.
  Undeclared external access from inside the platform is impossible, not discouraged —
  dbt's "no anonymous references," enforced at the network layer. Flagship behavior of
  the zero-trust guarantee family. No break-glass: rule 50 stands.
- **B3 — Guarantee attenuation at the wall: split by family.** Security-family
  guarantees REFUSE to compile across an external hop (no best-effort tier, rules
  37/50). Durability/freshness-family guarantees may compile across the wall as
  **labeled CONDITIONAL guarantees** with the boundary probe attached — the
  source-freshness analog: the claim is only as strong as the gate check, and the
  compiled output says so.
- **B4 — v1 scope: declare + deny; probes at skeleton.** v1 ships the `external`
  declaration and enforced default-deny egress with compiled allowlists. Boundary
  conformance probes (reachability/TLS/schema/freshness) arrive with the skeleton's
  verification plane, alongside attestation. An import/adoption ceremony (bringing an
  existing unmanaged resource inside, à la `terraform import`) is explicitly v2+.

### The `external` declaration (new primitive, binding)

d7s's `source()`: a named, schema-carrying declaration of a resource d7s did not compile
and will never provision or mutate. It is the **only legal crossing point** of the trust
boundary — all wiring to or from the outside references an `external` by name — and the
attachment point for boundary probes. Externals participate in the wiring graph and in
conditional guarantees; they never receive emitted infrastructure.

### Consequences

- The amended problem statement (Amendment 1) gains one sentence of force: guarantees
  are total inside the provenance boundary, conditional-and-labeled or refused across
  it — never silent.
- Week-one plan is unchanged (its default-deny slice already conforms); `external`
  declarations enter after week one, within v1.
- Open items: unchanged from Amendment 1, plus none added — B1–B4 are closed.

**Amendment status: recorded 2026-07-25; RE-SIGNED-OFF 2026-07-26, folded into the
Amendment 1 re-sign-off (see "Re-sign-off" section at the end of this document).**

---

## Re-sign-off — 2026-07-26

**Finding (flagged, not reconciled):** a prior commit (`1248528`, 2026-07-25) flipped
this document's top-line "Re-sign-off" status to SIGNED OFF without recording the
owner's statement verbatim, without updating the Amendment 1 and Amendment 2 status
footers (both still read "awaiting owner re-sign-off" as of that commit), and without
updating the dependent status lines in `../README.md` and `../../CLAUDE.md`. That edit
is superseded by the confirmation below, which completes the flip consistently across
all four locations in one commit.

### Owner statement (verbatim, 2026-07-26)

> Problem-definition re-sign-off CONFIRMED, covering Amendment 1 (guarantees-compiler)
> and Amendment 2 (trust-boundary). Flip the PENDING status header.

### Decision

Re-sign-off is **SIGNED OFF — 2026-07-26, by the owner**, covering Amendment 1
(guarantees-compiler reframing) and Amendment 2 (trust-boundary model) as one
confirmation, per the amendments' own terms (Amendment 1: "one explicit confirmation
that the amended problem statement... captures the intent"; Amendment 2: "folded into
the Amendment 1 re-sign-off"). The amended problem statement (end of Amendment 1) and
the trust-boundary model (Amendment 2) are the authoritative scope. No further scope
change accompanies this confirmation — it is a sign-off of what was already recorded,
not a new amendment.

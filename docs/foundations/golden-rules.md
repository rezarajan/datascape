# Golden rules

**Class: Contract.**

These are the rules distilled from the platformctl project (planning docs 00–12, ADRs
001–037, remediation and history records) that qualify as genuinely universal,
industry-proven practice for building production-grade software. Everything
platformctl-specific — its Kind vocabulary, capability roster, technology choices,
zero-trust plane, gate table — was deliberately left behind. What remains is the part
worth keeping regardless of what datascape turns out to be.

Each rule carries its industry precedent and its origin in the platformctl record
(`ptl:` references point into the platformctl repo — `docs/planning/NN`, `ADR NNN`).
Rules marked **⚠ paid-for** were learned through documented live failures, not just adopted
from literature.

---

## 1. Problem before solution

The rules platformctl *didn't* follow. They are first because their absence made every
other rule insufficient.

1. **Validate demand before building the mechanism.** A problem statement of the form
   "there is no tool that does X" is a hypothesis, not evidence. It requires named users,
   observed pain, and an analysis of why existing alternatives fail — before architecture.
   *Precedent:* Lean/continuous-discovery practice; every credible product methodology.
   *(ptl: 01-product-requirements §2 asserted the gap and never validated it — flagged in
   its own restart post-mortem.)* **⚠ paid-for**

2. **Success criteria must be outcome-facing, not mechanism-facing.** "Idempotent re-apply
   makes zero API calls" verifies the machine; "a new user gets a working pipeline in
   under N minutes and chooses the tool again" verifies the product. Ship criteria of both
   kinds; treat the outcome kind as the real gate.
   *Precedent:* SRE SLO practice; north-star/outcome metrics over output metrics.
   *(ptl: 01 §10 — all criteria were mechanism-facing.)* **⚠ paid-for**

3. **One primary persona per major version.** Multiple personas pulling in different
   directions (local-dev ergonomics vs CI determinism vs platform-substrate extensibility)
   let the heaviest architecture justify itself against the vaguest persona.
   *Precedent:* positioning/segment discipline (Moore, *Crossing the Chasm*).
   *(ptl: 01 §6 — the unnamed third persona quietly justified most of the weight.)* **⚠ paid-for**

4. **Re-validate the problem statement whenever scope grows.** Each scope accretion
   (object storage → catalogs → query engines → zero-trust networking) must re-answer
   "who asked?" against the original problem, or the problem statement must be formally
   revised. Silent scope growth is how a local-dev tool becomes an unshippable platform.
   *Precedent:* change-control over product intent, not just code.
   *(ptl: scope grew from 6 to 10 kinds, ~20 provider types, ~35 gates; the problem
   statement was never revisited.)* **⚠ paid-for**

5. **The definition of "production-ready" is fixed before the claim is made.** platformctl
   wrote five successive planning documents, each declaring a broader definition of
   production than the last, each audit finding the previous "complete" state short.
   Define the bar once, in writing, with checkable criteria — then the claim is a
   verification, not a negotiation.
   *Precedent:* Definition of Done; Kubernetes graduation criteria (KEPs).
   *(ptl: docs 07→08→09→11→12, documented in the post-mortem.)* **⚠ paid-for**

6. **Thin vertical slices; a walking skeleton from day one.** Every phase ships something
   runnable end-to-end — never "just the domain model" or "just the CLI skeleton."
   *Precedent:* walking skeleton (Cockburn/GOOS); tracer bullets (Pragmatic Programmer).
   *(ptl: 01 §5.6; this one was followed and worked.)*

7. **Declare known gaps explicitly rather than implying coverage.** "Behavior at 100s of
   resources is not yet characterized — a known, deliberate gap, not an implicit promise."
   Deferred items are noted so they aren't silently forgotten, not because they're promised.
   *Precedent:* honest-limitations sections in RFCs; Kubernetes scalability envelopes.
   *(ptl: 01 §8; ADR 004's "Known limitations (honest, not silently punted)" house norm.)*

## 2. Architecture

8. **Hexagonal layering with inward-pointing dependencies, enforced mechanically.**
   Domain imports nothing; ports import only domain; adapters implement ports; only the
   composition root knows concrete adapters. Enforced by an architecture test in CI —
   "discipline decays; nothing stops an import." This held through every platformctl audit
   and let a second runtime run every provider unmodified.
   *Precedent:* ports & adapters (Cockburn); ArchUnit/import-linter CI gates.
   *(ptl: ADR 008; 02-architecture §1–2.)*

9. **Name every architectural plane explicitly.** platformctl planned five of six
   production planes; the unnamed one (connectivity/discovery) is exactly where 10 of its
   17 live-caught defects clustered — "its logic precipitated into whichever provider
   needed it that day." An unnamed responsibility becomes a smeared responsibility.
   *Precedent:* control/data/management plane separation in networked systems.
   *(ptl: 09-systemic-findings §4; ADR 015.)* **⚠ paid-for**

10. **Keep the declarative model, the domain driver, and the execution substrate in
    separate layers.** The resource model must never know about the runtime; the runtime
    must never know domain semantics; exactly one layer understands both.
    *Precedent:* Kubernetes (API objects / controllers / kubelet); Terraform (config /
    providers / cloud APIs).
    *(ptl: 00-README "one diagram"; 02 §1.)*

11. **An abstraction with one implementation is a hypothesis; the second implementation is
    the test.** Prove every load-bearing seam with a real second implementation before
    freezing it — and expect the second implementation to find port-boundary defects that
    reasoning did not.
    *Precedent:* rule of three; Kubernetes CRI validated by multiple runtimes.
    *(ptl: 04 Phase 7; ADR 008 payoff; K2 VolumeSpec defect found only by building the
    Kubernetes adapter.)* **⚠ paid-for**

12. **Extension contracts are request structs, not accreting setter interfaces.** One
    request-scoped struct with additive fields: non-breaking for every implementor,
    stateless per call, serializable for a future wire protocol. Setter interfaces create
    temporal coupling the compiler can't check — platformctl deleted them as a defect.
    *Precedent:* Kubernetes admission.Request; protobuf additive-field evolution.
    *(ptl: ADR 016; 09 §3-F5.)* **⚠ paid-for**

13. **Capabilities are declared by small optional interfaces, checked in one place, at
    validate time.** Never a type explosion, never provider-type switches in the core,
    never a failure deferred to apply. The error message format is part of the contract.
    *Precedent:* Go optional interfaces (`io.ReaderFrom`); CSI/CNI capability discovery.
    *(ptl: ADR 009.)*

14. **Fix bug classes where reintroduction is impossible.** The same defect appearing at
    multiple call sites is a missing system mechanism, not N mistakes. The fix must live
    where a future author *cannot* reintroduce the class: the compiler refuses, the
    conformance suite refuses, or the capability isn't expressible. The test of an
    architectural fix: "could a new component, written today, reintroduce the bug?"
    *Precedent:* poka-yoke; "make illegal states unrepresentable."
    *(ptl: 09 §3 — ten call sites carried the identical hardcoded-address bug.)* **⚠ paid-for**

15. **Never construct what you can observe or resolve.** Addresses, runtime object names,
    network names: one authority publishes them; consumers resolve facts, never re-derive
    by convention. Killing the class in one vocabulary (addresses) does not kill it in the
    adjacent one (names) — audit the class, not the instance.
    *Precedent:* Kubernetes Services/DNS over hand-constructed pod IPs; service registries.
    *(ptl: ADR 015, ADR 030; 11 "decoupling verification".)* **⚠ paid-for**

16. **One naming authority mints all derived names.** Join rules, charset, length
    truncation with content-hash uniqueness, timestamp format — owned in one place, banned
    elsewhere by arch test. A name scheme proven on a lenient backend fails on a strict one.
    *Precedent:* Kubernetes generated-name/truncate-with-hash conventions; RFC 1123 as the
    strictest common denominator.
    *(ptl: ADR 030 — the authority's promise was already broken at seven sites before
    enforcement existed.)* **⚠ paid-for**

17. **Safety-critical guards live in exactly one enforcement point.** N per-component
    guards are N places to audit and one forgotten place from disaster.
    *Precedent:* reference-monitor concept in security design.
    *(ptl: ADR 013; 04 §14.)*

18. **Collapse redundant abstractions; resist premature generality.** Don't mimic another
    system's abstraction (storage classes, plugin protocols) before there's a second
    concrete need to abstract over. Keep speculative seams *shaped* for the future (frozen
    request struct, reserved enum values that refuse loudly) without building the machinery.
    Record concrete reopen criteria so "not now" is testable, not vibes.
    *Precedent:* YAGNI; rule of three; monolith-first.
    *(ptl: 00-README decision table; ADR 032.)*

19. **Wrappers over capability-bearing interfaces are a standing hazard.** A decorator
    that embeds an interface never promotes the optional capabilities of what it wraps.
    Every wrapper needs explicit delegation, pinned by a mechanical (reflective) test —
    platformctl was bitten three times before closing the class.
    *Precedent:* known decorator/interface-upgrade pitfall (Go community canon).
    *(ptl: ADR 018 addendum; 11 log.)* **⚠ paid-for**

## 3. Declarative model and reconciliation

20. **Desired state, with plan/apply separation.** A read-only plan always precedes apply;
    the plan is an ordered list of {resource, action, reason} consumed unchanged by apply.
    *Precedent:* Terraform plan/apply; Kubernetes dry-run.
    *(ptl: 01 FR-3/4; 02 §5.4–5.5.)*

21. **Idempotency is an interface contract, not a habit.** Methods are `Ensure*`, not
    `Create*`; a second call with an unchanged spec makes zero mutating calls — verified
    by conformance tests on every adapter.
    *Precedent:* Kubernetes reconciliation; HTTP PUT semantics.
    *(ptl: 02 §4.1; conformance suite.)*

22. **Determinism is a tested feature.** Same inputs → byte-identical plan, golden-file
    tested; live non-determinism is confined to status and never leaks into plan ordering
    or diffing.
    *Precedent:* Terraform plan stability; reproducible-build discipline.
    *(ptl: ADR 012; 02 §9.)*

23. **Drift is surfaced, never silently auto-corrected.** Only an explicit apply heals.
    And drift comparison is against desired *configuration*, not liveness — "RUNNING with
    the wrong topic" is drift.
    *Precedent:* Terraform refresh; Argo CD OutOfSync vs auto-sync distinction.
    *(ptl: 01 FR-8; 07 §2.1.)*

24. **Dependencies form a DAG; cycles are a hard error before any mutation; destroy walks
    the graph in reverse.** Per-resource failure halts dependents but continues
    independent branches. Watch for edges hiding in plain strings: a `host:port` field
    that references another resource without creating a graph edge produces arbitrary
    ordering that weak readiness checks will mask.
    *Precedent:* Terraform resource graph; systemd ordering.
    *(ptl: 01 FR-2/4/5; 11 log — the `Connection.spec.target` missing-edge finding.)* **⚠ paid-for**

25. **State is persisted atomically after each resource, versioned from day one, with
    advisory locking that names the holder.** A mid-run crash leaves state truthful;
    unrecognizable legacy state is refused, never guessed; locks have renewal (a lease
    that outlives its TTL mid-run is a corruption window) and stale-lock detection that
    reports recovery instructions.
    *Precedent:* Terraform state versioning/locking; WAL/atomic-rename idiom.
    *(ptl: ADR 012; ADR 003; 11 systems pass — the no-renewal lease finding.)* **⚠ paid-for**

26. **State records intent and published facts, never observations.** Persisting liveness
    lets status lie about things that died after the last apply.
    *Precedent:* spec/status separation (Kubernetes).
    *(ptl: ADR 017 kernel.)*

27. **Ownership labels on every created object; never touch anything unlabeled.** Refuse —
    never adopt — unmanaged same-name objects. Teardown operates on named objects only,
    never by pattern-matching live infrastructure state.
    *Precedent:* Kubernetes labels/ownerReferences; Terraform default tags.
    *(ptl: 02 §10; ADR 013; 11 — the over-broad teardown grep that deleted a user volume.)* **⚠ paid-for**

28. **Data-bearing resources default to retain-on-delete; destructive verbs need explicit
    double confirmation; a protect marker exists that no flag can waive.** And the
    protection path must be tested through the real end-to-end path — platformctl's
    `protect` field was inert from the day it shipped because tests bypassed the loader.
    *Precedent:* PV `Retain`; Terraform `prevent_destroy`; RDS deletion protection.
    *(ptl: ADR 013; 11 "latent safety bug".)* **⚠ paid-for**

29. **"Removed" is a stated contract: the object and everything derived from it are gone
    when the call returns — enforced by conformance.** An unstated cleanup contract lets an
    adapter pass every test while leaking on every call (platformctl leaked 3,853 volumes
    / 8.4 GB this way). Destructive semantics that differ between backends (refuse-in-use
    vs cascade-delete) are conformance-pinned on every adapter.
    *Precedent:* RAII/deterministic destruction; Kubernetes cascading deletion semantics.
    *(ptl: ADR 029; 07 cross-runtime — the RemoveNetwork namespace cascade.)* **⚠ paid-for**

30. **Exists ≠ ready ≠ reachable — three distinct states, codified in the contract.**
    "Ready means serving": answers its declared protocol right now, probed on the channel
    the consumer will actually use, with the same check drift uses. Adapters absorb async
    races; consumers never hand-roll retry loops.
    *Precedent:* Kubernetes readiness vs liveness; health-check-based load balancing.
    *(ptl: 09 Class 2; ADR 015 — pg_isready over a unix socket reported healthy before
    TCP listened.)* **⚠ paid-for**

31. **Distributed-system race classes must be hunted at every layer, not just where first
    observed.** An eventually-consistent view (e.g. metadata that differs between cluster
    members) will bite reconcile, probe, drift, and the test client separately; close the
    class everywhere or it recurs.
    *Precedent:* standard eventual-consistency engineering.
    *(ptl: 11 log — the heal-window class closed at four layers.)* **⚠ paid-for**

32. **Model relationships as relations, not functions, when future pairings are
    plausible; directionality lives in the edge, nouns stay role-neutral.** A
    one-pair-per-mode table makes tomorrow's pairing a breaking change; a relation makes
    it additive.
    *Precedent:* Kafka Connect's source/sink symmetry; additive API evolution.
    *(ptl: ADR 001 — the taxonomy revision that required owner intervention.)* **⚠ paid-for**

## 4. Validation and failure discipline

33. **Validate-time completeness.** A configuration set that validates must not be able to
    half-apply into a mis-wired system. Any configuration error first surfacing at apply
    is a *regression* whose fix is a new validation rule — never documentation telling
    users to be careful. Aggregate all failures into one report before touching anything.
    *Precedent:* shift-left; admission control; compiler-error philosophy.
    *(ptl: ADR 011.)*

34. **Fail fast and loudly on not-yet-implemented paths.** A schema-accepted field that
    nothing consumes is worse than a missing one — it manufactures false confidence
    (platformctl's `via` field was a silent security no-op). Reserved values refuse with
    "planned, not yet available."
    *Precedent:* HTTP 501; Kubernetes feature-gate rejection; Rust `todo!()` culture.
    *(ptl: 02 §4.4; 11 A1.)* **⚠ paid-for**

35. **Error messages carry the remedy.** Name what's wrong, what is actually supported,
    and what to do next. Timeout errors name the last observed state.
    *Precedent:* Rust/Elm diagnostics culture.
    *(ptl: 02 §5.2; 01 NFR-11.)*

36. **Optional integrations degrade to recorded, informational no-ops — never errors.**
    Declaring an optional hook a component doesn't consume must not block the component.
    (Security controls are the explicit exception — see rule 44.)
    *Precedent:* must-ignore extension semantics.
    *(ptl: 01 FR-20; ADR 010.)*

37. **A gate/flag interaction always gets an explicit fail-closed ruling.** When the
    feature interpreting a scoped declaration is off, the declaration is inert — a gate
    flip must never silently widen access or fall back to a broader behavior. A seam whose
    components are only partially landed stays off entirely: a resolver returning
    deterministic-but-unreachable addresses is dead code that looks alive.
    *Precedent:* fail-closed security design; dark-launch discipline.
    *(ptl: ADR 033 addendum; ADR 034 addendum.)* **⚠ paid-for**

## 5. Testing

38. **Contract/conformance suites per port, run against fakes AND real adapters.** This is
    what keeps the fake honest. And the fake is the *strictest* interpreter of the
    contract: the most permissive real backend must never define it — under-declared
    intent fails in fast tests, not on a cluster.
    *Precedent:* CSI/CRI conformance; Go `nettest.TestConn`; JDBC TCK.
    *(ptl: 02 §9; 09 §3-F2.)* **⚠ paid-for**

39. **The conformance ratchet: every live-caught bug lands with a contract-level
    reproduction in the same commit** — or is recorded as a per-backend difference. Without
    it, a third adapter rediscovers the bug the same expensive way.
    *Precedent:* regression-test-per-bugfix, made structural.
    *(ptl: 06 §8; ADR 015.)* **⚠ paid-for**

40. **Contract green is necessary, not sufficient: only real workloads on real
    infrastructure prove the translation.** platformctl's conformance was green while the
    Kubernetes adapter replaced every image's entrypoint — the synthetic fixture had no
    entrypoint to notice. Test fixtures must possess the property under test.
    *Precedent:* the e2e cap of the test pyramid, taken seriously.
    *(ptl: 07 cross-runtime, K1; 09 Class 5.)* **⚠ paid-for**

41. **One worked acceptance scenario is the executable definition of done** — docs example,
    CI e2e test, and release gate are the same artifact. And it must be exercised the way
    a user would invoke it: platformctl's flagship example was un-loadable by the CLI for
    the project's entire life because nobody ran the documented invocation.
    *Precedent:* executable specifications; acceptance-test-driven development.
    *(ptl: 05 §6–9; 08 §10 M7 BLOCKED.)* **⚠ paid-for**

42. **N green features ≠ a green product: composition gets its own acceptance tests.** Two
    individually-correct features can be mutually destructive (platformctl's default-deny
    isolation silently blocked its own access-mode feature).
    *Precedent:* integration/interaction testing discipline.
    *(ptl: history/errors.md B7 RCA; 11 capstone finding.)* **⚠ paid-for**

43. **Test tiering with CI as arbiter.** Fast tier (fakes, strict time budget enforced by a
    CI guard — a slow fast-test is a defect) is the only thing a developer waits for; deep
    tier is explicit and impact-selected; stress runs in CI. Local green is a signal; CI
    green is the verdict — warm local environments hide the races CI's cold runners expose.
    *Precedent:* Google small/medium/large taxonomy.
    *(ptl: ADR 028.)*

44. **No fixed-duration sleeps, anywhere.** Poll observable conditions under an honest
    deadline; deadlines bound failure reporting, never success; one environment knob
    scales all waits rather than anyone guessing bigger constants.
    *Precedent:* Kubernetes `wait.Poll`; universal flaky-test guidance.
    *(ptl: 01 NFR-11; 11 timed-poll census.)*

45. **Golden-file tests for output stability; a byte-stable machine-output contract**
    (exactly one parseable document on stdout per command per exit path, prose to stderr,
    documented deterministic exit codes) — verified by a harness, not by inspection.
    *Precedent:* POSIX conventions; snapshot testing; git/terraform exit-code contracts.
    *(ptl: 02 §6–9; 07 §0.5; remediation F-001.)* **⚠ paid-for**

46. **Test hostile inputs against the real parsers, and test safety paths through the real
    end-to-end path.** Credentials contain `@ : / # spaces quotes backslashes` — round-trip
    them through the actual drivers; never rely on lucky demo passwords. Safety features
    verified only via hand-constructed internal values (bypassing the real
    loader/parser) can be inert in production while every test stays green.
    *Precedent:* adversarial input testing; property-based testing.
    *(ptl: 07 §2.2; 11 — shell injection via manifest field; the inert `protect` field.)* **⚠ paid-for**

47. **Every failure investigation validates root cause by reproduction, and records
    disproven hypotheses** so future audits don't re-plow the same ground. Regression
    evidence beats plausible narrative — platformctl's first RCA of its isolation bug was
    wrong and was only corrected by direct reproduction.
    *Precedent:* blameless post-mortem practice; scientific method applied to debugging.
    *(ptl: history/errors.md corrected RCA.)* **⚠ paid-for**

48. **Silent success is the deadliest failure mode: verify the artifact, not the exit
    code.** platformctl's backup pipeline reported success through five layers while
    producing a 0-byte file with a matching empty checksum. Verify-then-promote: unverified
    data never reaches the live target — stream to scratch, checksum the exact bytes that
    crossed the pipe, promote atomically; on failure the target was never touched.
    *Precedent:* database-engineering restore-verification doctrine ("a backup is not a
    backup until it has been restored").
    *(ptl: 11 I15; ADR 007.)* **⚠ paid-for**

## 6. Security

49. **Security is proven by the negative probe, never assumed.** An unverifiable claim is
    treated as false. Denied paths must fail a live reachability probe executed from the
    real consumer's vantage point; status reports what was observed (`enforced` /
    `NOT ENFORCED` / `unknown`); a skipped enforcement test must never masquerade as
    coverage (platformctl's CI ran a CNI that never enforced NetworkPolicy — every
    isolation assertion silently skipped for the project's life).
    *Precedent:* zero-trust doctrine; NIST 800-207; adversarial verification.
    *(ptl: ADR 027; 11 GA caveat sweep.)* **⚠ paid-for**

50. **Security properties have no best-effort tier.** An unenforced access restriction is
    not "degraded" — it is wrong, and must be refused. Never delegate a security guarantee
    to an optional substrate feature; a guarantee that varies by substrate is not a
    guarantee. Identity is authoritative; network segmentation is defense-in-depth.
    *Precedent:* fail-closed doctrine; SPIFFE/BeyondCorp layering.
    *(ptl: ADR 037 kernel; ADR 027.)* **⚠ paid-for**

51. **Secrets are references, never inline values — unrepresentable at the schema level.**
    State stores one-way fingerprints only; secret values never appear in plans, logs, or
    state; rotation is detected by fingerprint drift and cascaded to dependents. Secret
    material rides file mounts with tight permissions, never env vars or argv. Recovery
    boundaries for rotation are documented honestly, with rejected alternatives recorded.
    *Precedent:* 12-factor config separation; Vault/ESO patterns.
    *(ptl: 03 §10; ADR 007.)*

52. **Policy loads from a channel outside the set it governs** — the governed set must not
    be able to amend its own guardrails. Deny wins; exemptions only if the policy declares
    itself exemptible; user policy may only *intersect* an auto-derived least-privilege
    baseline, never widen it.
    *Precedent:* separation of duties; AWS SCPs; OPA/Gatekeeper.
    *(ptl: ADR 021; ADR 035 kernel d.)*

53. **Least privilege compiles from the declared dependency graph.** In a system where
    every dependency is a declared, reviewed reference, the reference graph *is* the
    access-request set — no second permission language for the common case; wide grants
    are explicit, reviewable declarations. Labels/claims are written by whoever wants
    access, so who may *wear* a claim must be as governable as who may require one.
    *Precedent:* capability-based security ("no ambient authority"); service-mesh intentions.
    *(ptl: ADR 026; ADR 033.)*

54. **Withdrawal of permission means refusal at next admission, never automatic teardown
    of live infrastructure.** A destroy that no plan proposed and no human approved is a
    new unaudited actor. The gap between policy and reality is a reported state.
    *Precedent:* GitOps "config change ≠ silent destructive convergence."
    *(ptl: ADR 021 amendment.)*

## 7. Release engineering and evolution

55. **Gate, don't branch.** Every new capability ships behind a named gate, disabled by
    default, in the release it's built; main is always releasable. "Gate off = zero
    behavior change" is a behavioral contract — byte-identical output, not "mostly none."
    Gate names are API. Defaults are risk-proportional: new attack/blast surface defaults
    off until soaked, with reasoning recorded.
    *Precedent:* Kubernetes feature gates; trunk-based development.
    *(ptl: ADR 014.)*

56. **Maturity stages carry written, checkable commitments** (shape stability, default
    state, minimum evidence), declared at introduction, with graduation triggers named
    up front. No GA without conclusive evidence; recorded deviations are debts, not
    resolutions. Track lapsed graduations — they accumulate silently.
    *Precedent:* Kubernetes API deprecation policy and feature stages.
    *(ptl: 04 §12.1; 11 owner rulings; ADR 014 lesson.)*

57. **ADR discipline: settled decisions, immutable records, one decision each.** Large
    tasks write the design note first. Changing a decision starts a new ADR that names
    what it amends; superseded first cuts are kept as history. When an ADR and a contract
    doc disagree, the contract doc wins, updated in the same commit.
    *Precedent:* Nygard ADRs, as practiced by every mature platform team.
    *(ptl: adr/README; 08 §2.1.)*

58. **Exit criteria are verified against the running artifact, never from memory.** "Done"
    means acceptance criteria *ran*, not "would pass if run." Audit checkbox truth — status
    claims drift from code in both directions; mechanize sync where possible (generated
    docs in-sync tests, README-surface tests).
    *Precedent:* Definition of Done; docs-as-code drift gates.
    *(ptl: 04 exit criteria; remediation F-001/F-006 — a checked box that was false.)* **⚠ paid-for**

59. **Every deferral carries a reason and exactly one home.** Nothing is silently dropped:
    mapped to a task, deferred with rationale, or designated permanently-unsupported.
    Deviations from plan are findings for a maintainer decision, never silent scope changes.
    *Precedent:* change-control discipline; RFC errata culture.
    *(ptl: 08 §9; 06 §1.)*

60. **Upgrade notes exist only for changes that would otherwise look like regressions**,
    in a fixed shape: Affects / What changed / What you'll see / What it does NOT do /
    Action required. Release checklists are executable by someone (or something) with no
    context beyond the checklist.
    *Precedent:* keep-a-changelog discipline, sharpened; runbook culture.
    *(ptl: upgrade-notes.md; releasing.md.)*

61. **Prefer a dependency class you already carry over a technically superior new one.**
    A new dependency class is an ongoing tax; "disproportionate for one JSON file plus a
    lock" generalizes. Similarly: don't rebuild operationally-deep products (consensus,
    failover, backup engines) inside your tool — integrate them at a declared seam, and
    when saying "not yet," enumerate what a reversal would require.
    *Precedent:* dependency-minimalism (Go proverb culture); buy-vs-build discipline.
    *(ptl: ADR 003, cited as precedent by four later ADRs; ADR 005.)*

62. **Choose infrastructure components by fit with your operational model.** platformctl
    chose its proxy solely because its admin API was read-write and object-addressable —
    matching reconcile-by-idempotent-API-call; a component configured by
    "rewrite the file and restart" cannot be a shared multi-tenant piece, because
    per-tenant changes must never restart the shared component.
    *Precedent:* operational-fit selection (the reason etcd, Envoy, and Caddy win their
    niches).
    *(ptl: ADR 018 kernels.)*

## 8. Operability and developer experience

63. **Conditions are the health surface, not log-scraping; one condition answers "is it
    working."** Structured, machine-parseable events for every action — and "structured"
    is a parse-tested contract, not a metaphor.
    *Precedent:* Kubernetes status.conditions (KEP-1623); structured-logging practice.
    *(ptl: 02 §3.7, §8; 01 NFR-12.)*

64. **Library and adapter code never writes to process-global streams.** An injectable
    diagnostics channel carries warnings; the severity ladder is closed at the contract
    level (fail / degrade / inform) — components may not invent a fourth channel.
    *Precedent:* "libraries don't print" doctrine.
    *(ptl: ADR 031.)*

65. **Convention over configuration, with escape hatches, deterministically derived.**
    Defaults are derived deterministically (same input → same default, no collisions —
    and deterministic hashing into a small space needs collision detection with a named
    remedy); explicit values always win; per-technology default profiles bound otherwise
    undecorated workloads.
    *Precedent:* Rails/Go-modules convention culture; Kubernetes defaulting.
    *(ptl: ADR 035; 11 systems pass — the birthday-collision finding.)* **⚠ paid-for**

66. **Generators compile to the declarative source of truth; they never bypass it.**
    Scaffolding writes files; the normal validate/plan/apply pipeline is unchanged, so
    every guardrail applies automatically and output is diffable, reviewable, GitOps-safe.
    Generated output meets the same quality bar as hand-written.
    *Precedent:* `rails generate` / `kubebuilder`; GitOps.
    *(ptl: ADR 024.)*

67. **Reference docs are generated from the source of truth and CI-checked for drift;
    docs and schemas change in the same commit.** Stale docs are architectural debt —
    contributors copy the wrong patterns. Docs show actual error text, not paraphrases,
    and every diagnostic code has a generated explanation (`explain <token>`) built from
    the same constants the code uses.
    *Precedent:* docs-as-code; OpenAPI-generated references.
    *(ptl: docs/reference pipeline; 07 §3.3; explain catalog.)*

68. **Classify every document as contract, plan, or record — and enforce the
    classification.** Contracts are what code is checked against; plans evolve additively;
    records are append-only ("append facts, never revise meaning"). One index page
    declares every document's class.
    *Precedent:* document-control practice from regulated engineering, profitably
    lightweight here.
    *(ptl: docs/README; the planning-doc guard hook.)*

69. **Position honestly against incumbents, including "when to use the other tool."**
    Tie every capability claim to a named test, ADR, or file; fence future ideas as "not
    scheduled work"; keep the maturity row honest against yourself. Claims that outrun
    citations are the tell.
    *Precedent:* honest-comparison marketing (the kind engineers trust).
    *(ptl: positioning/terraform.md — both its discipline and its one weakly-sourced claim.)*

70. **Three-tier naming: product name (prose), binary name (what users type), identifier
    stem (wire/disk/env).** Identifier surfaces are frozen compatibility contracts a
    branding pass may never cross casually; brand aliases live in prose only.
    *Precedent:* the Kubernetes/k8s/kubectl split, legislated instead of accidental.
    *(ptl: ADR 019 — written in anticipation of exactly this restart.)*

---

## How to use this document

- During discovery (now): rules 1–7 are the active contract; the rest wait.
  *(Superseded 2026-07-25 — see the amendment below: solution work is authorized and
  every un-struck rule binds.)*
- When solution work starts: this document is reviewed once against the defined problem —
  rules that don't apply to what datascape becomes are struck with a recorded reason, not
  silently ignored. What remains binds.
- A rule is amended the way an ADR is: a new dated entry naming what it changes and why.

---

## Amendment — 2026-07-25: v1 applicability review (problem definition Q3.4)

**What this amends and why.** The one-time review prescribed above was performed against
the signed-off problem definition (`../discovery/00-problem-definition.md`, SIGNED OFF
2026-07-25). The shape reviewed against: datascape is a **GitOps compiler** — it compiles
stack declarations into Flux-consumable manifests for one Kubernetes target; it owns no
reconcile loop, no mutating verbs against any backend, and no state store (git is the
state; the cluster belongs to Flux and the operators). Zero-trust is the differentiator;
verifiable compute = supply-chain attestation + policy admission proofs +
declared=running; the lakehouse deployment is the acceptance workload; multi-runtime
and day-2 operations are refused.

**Outcome: 67 of 70 rules bind. Rules 21, 25, and 26 are struck** — all three for the
same structural reason: they constrain a mutating runtime engine and an owned state
store, and a GitOps compiler has neither. Struck rules keep their text above; each
carries a reopen criterion here.

### Struck rules

- **Rule 21 (idempotency as an `Ensure*` interface contract) — struck.** Datascape makes
  zero mutating calls against any backend, so there are no `Ensure*` adapters for the
  contract to bind. The property the rule protected — a re-run is safe — is carried by
  rule 22 (same declaration → byte-identical output) and by Flux's own idempotent
  reconciliation. *Reopens automatically* if datascape ever gains a mutating adapter —
  a direct-apply mode, a bootstrap verb, anything that talks to a cluster API with
  write intent.

- **Rule 25 (atomic state persistence, versioning, advisory locking) — struck.**
  Datascape owns no state store: compiled output in git is the only state it produces,
  and git already provides atomicity, versioning, and history. Cluster state belongs to
  Flux and the operators. *Reopens* the moment datascape persists anything beyond the
  git worktree — a compile cache, a verification ledger, a lock file.

- **Rule 26 (state records intent, never observations) — struck**, same reason as
  rule 25: there is no state store to constrain. Its spirit survives in one binding
  place: compiled output and its attestations record intent only; observations appear
  solely in declared=running verification *reports*, which are never persisted as
  authority. Same reopen criterion as rule 25.

### Binding interpretations (rule text unchanged)

How specific rules bind in the compiler shape — recorded once so they aren't
re-litigated per task:

- **Rule 11 (second implementation tests the seam):** binds, carrying the Q3.3 flag:
  the Flux emitter interface has one implementation and is therefore a hypothesis, kept
  exactly as thin as the compiler's output boundary (emit-manifests-for-target).
  Nothing is designed for a hypothetical second target until one genuinely arrives.
- **Rule 20 (plan/apply separation):** compile is the plan — read-only and
  deterministic; the git diff of compiled output is the reviewable plan artifact; apply
  belongs to Flux and is triggered only by merge. No datascape verb mutates a cluster.
- **Rule 23 (drift surfaced, never silently auto-corrected):** datascape's
  declared=running verification only reports; it never mutates. Flux healing drift is
  not a violation of this rule — it is the explicitly declared, configured behavior of
  the reconciler the user chose, not silent auto-correction. Drift comparison remains
  against desired configuration, not liveness.
- **Rule 24 (dependency DAG):** cycles are a compile-time hard error; ordering edges
  compile into the target's dependency semantics (Flux `dependsOn`); the
  hidden-edge-in-a-string clause binds in full — wiring fields (`host:port`, connection
  URIs) must create graph edges, because cross-component wiring is the product's
  core job.
- **Rule 27 (ownership labels):** every compiled object carries datascape ownership
  labels; Flux prune scope and declared=running verification operate only on
  labeled objects.
- **Rule 29 ("removed" is a stated contract):** removal becomes a compile-time
  contract — what removing a component from the declaration causes (pruned vs
  retained, per component class) is stated and conformance-tested on compiled output;
  data-bearing components compose with rule 28's retain-by-default.
- **Rule 30 (exists ≠ ready ≠ reachable):** binds on the verification and acceptance
  surface — probes run from the real consumer's vantage point, over the channel the
  consumer will actually use (through the mesh, with mTLS), not inside datascape's
  absent runtime loop.
- **Rule 63 (conditions as the health surface):** runtime conditions belong to the
  operators and Flux; declared=running *reads* those conditions rather than re-deriving
  health, and datascape's own CLI/verification output is the structured, parse-tested
  surface.

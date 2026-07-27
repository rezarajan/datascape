package flux

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/rezarajan/datascape/internal/domain"
)

// This file compiles the egress guarantee family (week-four plan, slice
// 1; problem definition Amendment 2, B2: "Egress is compiled
// default-deny; allowlists come only from declared wiring" — a contract
// requirement week-two and week-three both deferred, superseded here).
// Two declared-wiring sources exist this week: a self-hosted component's
// guarantees.rpo.backupTo naming a declared external (its backup traffic
// leaving the mesh), and a managed component's declared allowedConsumers
// (its consumers' traffic leaving the mesh toward the Neon endpoint).
// Both compile to the same upstream mechanism, verified against current
// Istio docs rather than memory (2026-07-26):
//
//   - istio.io/latest/docs/reference/config/security/authorization-policy/
//     documents AuthorizationPolicy.spec.targetRefs supporting
//     `kind: ServiceEntry, group: networking.istio.io` in the policy's own
//     namespace — the attachment shape this file uses, since a declared
//     external endpoint is not an in-mesh workload a selector could match
//     (mtls.go's existing selector-based policies only ever gate
//     workloads d7s itself compiled, e.g. the CNPG Cluster's pods).
//   - ambientmesh.io's waypoint-authz and mesh-egress docs (fetched
//     2026-07-26; verified quote: "Without a waypoint installed, you can
//     only use Layer 4 security policies" and selector-based policies
//     are "enforced by ztunnel") establish that a targetRefs-attached
//     policy is a WAYPOINT mechanism, not a ztunnel-alone one — ztunnel
//     has nothing at the destination side of a ServiceEntry to enforce
//     against. So every ServiceEntry this file emits carries
//     `istio.io/use-waypoint`, and every namespace that needs one gets
//     exactly one waypoint (emitWaypoint) — verified against
//     istio.io/latest/docs/ambient/usage/waypoint/ for the Gateway API
//     shape (gatewayClassName: istio-waypoint) and ambientmesh.io's
//     troubleshooting doc for the ServiceEntry label shape.
//
// This is the honest design the plan asked for, not an assumed one: if
// ambient's own model required something d7s cannot compile at all, this
// file would refuse instead of guessing — it does not need to, because
// the waypoint-attached ServiceEntry/AuthorizationPolicy path is a
// documented, supported upstream shape for exactly this problem
// (egress authorization to a destination with no in-mesh workload of its
// own).
const (
	// waypointName is fixed for v1 — one shared waypoint per stack
	// namespace, the same "one object, whichever guarantees need it"
	// discipline emitCNPGOperator already follows for the operator
	// install.
	waypointName             = "waypoint"
	waypointGatewayClassName = "istio-waypoint"
	waypointForLabel         = "istio.io/waypoint-for"
	waypointForService       = "service"
	waypointListenerName     = "mesh"
	waypointListenerPort     = 15008
	waypointListenerProto    = "HBONE"

	// useWaypointLabel is the label istiod looks for (in the same
	// namespace, verified against istio.io/latest/docs/ambient/usage/
	// waypoint/, 2026-07-26) to bind a ServiceEntry's traffic through a
	// named waypoint.
	useWaypointLabel = "istio.io/use-waypoint"

	// serviceEntryTargetRefGroup and serviceEntryTargetRefKind are the
	// fixed targetRefs attachment naming an emitted AuthorizationPolicy
	// uses to attach to its ServiceEntry (verified against istio.io's
	// AuthorizationPolicy reference, 2026-07-26).
	serviceEntryTargetRefGroup = "networking.istio.io"
	serviceEntryTargetRefKind  = "ServiceEntry"

	// neonPostgresPort is Neon's connection proxy's fixed listening port
	// (verified against Neon's own connection docs, 2026-07-26 — the
	// provider exposes no separate port attribute because there is only
	// ever one, the same fact neonConfigTemplate's "port" output in
	// terraform.go hard-codes).
	//
	// SUPERSEDED DESIGN, recorded not deleted (this constant used to sit
	// beside a neonWildcardHost = "*.neon.tech" constant and a
	// domain-pattern ServiceEntry design — the honest compile-time limit
	// this file originally chose because the Neon endpoint host a
	// managed component's Terraform CR provisions is NOT known at
	// compile time, only after tofu-controller reconciles and writes it
	// to the credentials Secret (terraform.go's
	// TerraformWriteOutputsToSecret), and d7s never reads that Secret
	// itself (rule 51)):
	//
	// 2026-07-27 finding (week-four plan → Revision 2): the domain-
	// pattern design does not actually work. Istio 1.30 ambient
	// waypoints do not program wildcard TLS ServiceEntries — the
	// capability (wildcard host + DYNAMIC_DNS resolution) is alpha,
	// gated behind istiod's mesh-wide default-off
	// ENABLE_WILDCARD_HOST_SERVICE_ENTRIES_FOR_TLS flag, which upstream
	// itself marks "not production ready, susceptible to SNI spoofing,
	// trusted clients only." A live waypoint was proven to refuse to
	// route to the wildcard host under both Postgres TLS negotiation
	// modes (legacy and sslnegotiation=direct) — confirmed route-
	// absence, not an SNI-timing artifact (which the file's earlier
	// comment here had flagged as the open empirical question; it
	// wasn't the cause). The owner declined enabling the alpha flag —
	// "no alpha flag" — consistent with golden rule 50 (no best-effort
	// security tier): a flag upstream itself calls not production-ready
	// is not a foundation for a compiled security guarantee. Revisit
	// only if/when upstream graduates the feature.
	//
	// The design now is EXACT-HOST PINNING: domain.Postgres.EndpointHost
	// (internal/domain/postgres.go) carries the operator-supplied exact
	// endpoint host, read from the writeOutputsToSecret Secret after
	// first provisioning. allowedConsumers on a managed component
	// without a pin refuses at domain validation, naming the two-step
	// ceremony (provision unpinned, deliver, read the host, pin,
	// recompile, redeliver) — no compiled artifact ever names an
	// unproven host. emitNeonEgress below now emits an EXACT-host
	// ServiceEntry from pg.EndpointHost — the same proven shape
	// neonControlPlaneHost's own console.neon.tech:443 entry already
	// uses (plain DNS resolution, no DYNAMIC_DNS), strictly MORE precise
	// than the retired wildcard's domain-wide scope, not less.
	neonPostgresPort = 5432

	// neonControlPlaneName, neonControlPlaneHost, and
	// neonControlPlanePort compile the provisioner's own edge — a live
	// composition bug this egress enforcement caught (2026-07-26): the
	// managed scenario's tf-runner pod, now correctly mesh-captured once
	// any managed component makes the stack namespace ambient, failed
	// `terraform plan` with `Get "https://console.neon.tech/api/v2/...":
	// EOF` — console.neon.tech matches the *.neon.tech ServiceEntry's
	// HOST wildcard, but that ServiceEntry only lists port 5432, so the
	// tf-runner's HTTPS/443 control-plane call to Neon's API had no
	// matching port and was denied. The zero-trust posture was doing its
	// job; the declared wiring graph was incomplete — placement: managed
	// itself implies this edge (every managed component's Terraform CR
	// calls Neon's API to provision it, unconditionally, regardless of
	// whether that component also declares allowedConsumers), so it is
	// declared wiring, not a new declaration the schema needs.
	//
	// Unlike the per-branch data-plane endpoint (domain.Postgres.
	// EndpointHost — genuinely runtime-bound, only known after
	// provisioning), the control-plane host is NOT runtime-bound: it is
	// the kislerdm/neon provider's fixed API base URL,
	// https://console.neon.tech/api/v2 (verified against the provider's
	// own docs and Neon's Terraform guide, 2026-07-26 — "cf. the SDK's
	// baseURL"). This dedicated, EXACT-host ServiceEntry was, and
	// remains, simpler and more honest than the now-retired wildcard
	// design ever was: it never needs DYNAMIC_DNS, and it never widens
	// what a data-plane consumer is allowed to reach. Compiled once per
	// stack (like emitWaypoint
	// and emitManagedRunnerRBAC) rather than once per managed
	// component: the tf-runner ServiceAccount is itself shared across
	// every managed component in a stack (emitManagedRunnerRBAC), so a
	// second identical ServiceEntry/AuthorizationPolicy pair per
	// component would be redundant, not more precise.
	neonControlPlaneName = "neon-control-plane"
	neonControlPlaneHost = "console.neon.tech"
	neonControlPlanePort = 443
)

// Gateway is the subset of a Kubernetes Gateway API
// gateway.networking.k8s.io Gateway d7s emits — used only to deploy an
// Istio ambient waypoint proxy. gatewayClassName: istio-waypoint is what
// tells istiod to provision and manage the waypoint deployment/service
// behind this object (verified against istio.io/latest/docs/ambient/
// usage/waypoint/, 2026-07-26). The Gateway API's own CRDs are a
// declared environment prerequisite (alongside Istio ambient itself),
// not something this compiler installs.
type Gateway struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   ObjectMeta  `yaml:"metadata"`
	Spec       GatewaySpec `yaml:"spec"`
}

// GatewaySpec is the subset of Gateway.spec d7s emits.
type GatewaySpec struct {
	GatewayClassName string            `yaml:"gatewayClassName"`
	Listeners        []GatewayListener `yaml:"listeners"`
}

// GatewayListener is one entry of Gateway.spec.listeners. The
// mesh/15008/HBONE listener is the fixed shape every ambient waypoint
// declares (verified against istio.io/latest/docs/ambient/usage/
// waypoint/, 2026-07-26) — not a d7s choice, Istio's own convention.
type GatewayListener struct {
	Name     string `yaml:"name"`
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

// ServiceEntry is the subset of an Istio networking.istio.io ServiceEntry
// d7s emits: the declaration that a destination outside the mesh's own
// service registry exists and may be routed to at all — the first half
// of egress compilation's declare+deny pair (Amendment 2, B2). Location
// is always MESH_EXTERNAL (every host this emits is, by construction, a
// declared external or a managed provider's own endpoint — never
// something d7s itself provisioned into the mesh's service registry).
type ServiceEntry struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   ObjectMeta       `yaml:"metadata"`
	Spec       ServiceEntrySpec `yaml:"spec"`
}

// ServiceEntrySpec is the subset of ServiceEntry.spec d7s emits.
type ServiceEntrySpec struct {
	Hosts      []string           `yaml:"hosts"`
	Location   string             `yaml:"location"`
	Ports      []ServiceEntryPort `yaml:"ports"`
	Resolution string             `yaml:"resolution"`
}

// ServiceEntryPort is one entry of ServiceEntry.spec.ports.
type ServiceEntryPort struct {
	Number   int    `yaml:"number"`
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
}

// backupEgressTarget is one declared external a stack's self-hosted
// components back up to, resolved to the host/port a ServiceEntry needs
// plus every consuming component's name (each backed by CNPG's own
// per-Cluster ServiceAccount, verified against CloudNativePG's default
// behavior, 2026-07-26: a Cluster named "orders-db" gets a ServiceAccount
// also named "orders-db" in the same namespace) — the declared backup
// wiring IS the allow-list (plan slice 1), nothing wider.
type backupEgressTarget struct {
	ExternalName string
	Host         string
	Port         int
	Consumers    []string
}

// backupEgressTargets gathers one target per declared external actually
// referenced by a self-hosted component's guarantees.rpo.backupTo,
// deduplicating by external name so two components backing up to the
// same external share one compiled ServiceEntry with both principals
// allowed, rather than two competing declarations of the same host.
// Built by walking selfHosted in its own (deterministic, declaration-
// order) sequence — no map iteration ever decides output order, so
// output stays byte-identical across compiles (golden rules 22, 45).
//
// Every problem found across the whole loop is collected and joined
// (golden rule 33: validate-time completeness) rather than returned on
// the first one — mirrors Emit's own first loop over stack.Components,
// which does the same (var errs []error, append, continue, then
// errors.Join once at the end) rather than stopping at the first bad
// component.
func backupEgressTargets(selfHosted []domain.Postgres, externals map[string]domain.External) ([]backupEgressTarget, error) {
	byName := make(map[string]*backupEgressTarget)
	var order []string
	var errs []error
	for _, pg := range selfHosted {
		if pg.Guarantees.RPO == nil {
			continue
		}
		name := pg.Guarantees.RPO.BackupTo
		t, seen := byName[name]
		if !seen {
			ext, ok := externals[name]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"flux emitter: postgres component %q: guarantees.rpo.backupTo %q does not resolve to a declared external — this is a defect (domain validation should have caught it)",
					pg.Name, name))
				continue
			}
			host, port, err := objectStoreHostPort(ext)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			t = &backupEgressTarget{ExternalName: name, Host: host, Port: port}
			byName[name] = t
			order = append(order, name)
		}
		t.Consumers = append(t.Consumers, pg.Name)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	targets := make([]backupEgressTarget, 0, len(order))
	for _, name := range order {
		targets = append(targets, *byName[name])
	}
	return targets, nil
}

// objectStoreHostPort resolves a declared external object store's
// endpoint into the host and port a ServiceEntry needs. This is egress
// compilation's own compile-time check (plan slice 1, "refuses what
// declared wiring doesn't cover, with remedies"): CNPG's barmanObjectStore
// (durability.go) accepts the endpoint as a full URL verbatim, but a
// ServiceEntry needs a bare host and an explicit port — an endpoint
// lacking one cannot honestly compile a scoped allow-list, so it refuses
// here (golden rules 34, 35) rather than guessing a default port.
func objectStoreHostPort(ext domain.External) (host string, port int, err error) {
	if ext.ObjectStore == nil {
		return "", 0, fmt.Errorf(
			"flux emitter: external %q has no objectStore declared — this is a defect (domain validation should have caught it)",
			ext.Name)
	}
	endpoint := ext.ObjectStore.Endpoint
	u, perr := url.Parse(endpoint)
	if perr != nil || u.Hostname() == "" {
		return "", 0, fmt.Errorf(
			"flux emitter: external %q: objectStore.endpoint %q cannot compile an egress ServiceEntry — "+
				"it must be a URL with an explicit host (e.g. https://host:9000); fix the declared endpoint",
			ext.Name, endpoint)
	}
	portStr := u.Port()
	if portStr == "" {
		return "", 0, fmt.Errorf(
			"flux emitter: external %q: objectStore.endpoint %q has no explicit port — "+
				"egress compilation needs one to scope the ServiceEntry (e.g. https://host:9000); "+
				"add a port to the declared endpoint",
			ext.Name, endpoint)
	}
	p, aerr := strconv.Atoi(portStr)
	if aerr != nil {
		return "", 0, fmt.Errorf(
			"flux emitter: external %q: objectStore.endpoint %q has a non-numeric port %q",
			ext.Name, endpoint, portStr)
	}
	return u.Hostname(), p, nil
}

// neonEgressTargets returns every managed Postgres component that
// declares allowedConsumers — the ones whose DATA-PLANE branch endpoint
// egress compilation covers (week-four plan, slice 2's un-refusal). A
// managed component with no declared consumers stays reachable by
// nothing on its data plane (the correct default-deny state, mirroring
// AllowedConsumer's own doc comment) and compiles no per-component
// ServiceEntry/AuthorizationPolicy — presence is still the only signal
// (golden rule 50). This is independent of the CONTROL-PLANE edge
// (emitNeonControlPlaneEgress), which every managed component needs
// regardless of declared consumers — see neonControlPlaneHost's doc
// comment for the live-caught bug this distinction closes.
func neonEgressTargets(managed []domain.Postgres) []domain.Postgres {
	var out []domain.Postgres
	for _, pg := range managed {
		if len(pg.AllowedConsumers) > 0 {
			out = append(out, pg)
		}
	}
	return out
}

// emitEgress compiles the egress guarantee family for stack (week-four
// plan, slice 1+2): nothing at all when no component declares wiring
// that crosses the mesh boundary; otherwise exactly one shared waypoint,
// one ServiceEntry/AuthorizationPolicy pair per declared external and per
// managed component with declared consumers, and — unconditionally for
// every managed component, declared consumers or not — the provisioner's
// own control-plane edge (neonControlPlaneHost's doc comment: a managed
// component's Terraform CR always calls Neon's API to provision it, so
// that edge is implied by placement: managed itself, not by
// allowedConsumers).
func emitEgress(files map[string][]byte, stackName string, selfHosted, managed []domain.Postgres, externalsByName map[string]domain.External) error {
	backupTargets, err := backupEgressTargets(selfHosted, externalsByName)
	if err != nil {
		return err
	}
	neonTargets := neonEgressTargets(managed)
	needsControlPlane := len(managed) > 0

	if len(backupTargets) == 0 && len(neonTargets) == 0 && !needsControlPlane {
		return nil
	}

	if err := emitWaypoint(files, stackName); err != nil {
		return err
	}
	for _, t := range backupTargets {
		if err := emitBackupEgress(files, stackName, t); err != nil {
			return err
		}
	}
	for _, pg := range neonTargets {
		if err := emitNeonEgress(files, stackName, pg); err != nil {
			return err
		}
	}
	if needsControlPlane {
		if err := emitNeonControlPlaneEgress(files, stackName); err != nil {
			return err
		}
	}
	return nil
}

// emitWaypoint compiles the one waypoint proxy a stack namespace needs
// to enforce any egress ServiceEntry's AuthorizationPolicy (see this
// file's top-of-file doc comment for why a waypoint is required at all:
// ztunnel alone only enforces selector-attached, in-mesh-workload
// policies). Idempotent — safe to call once per stack regardless of how
// many egress targets exist, the same convention emitCNPGOperator
// already follows for its own shared install.
func emitWaypoint(files map[string][]byte, stackName string) error {
	labels := ownershipLabels(stackName, "")
	labels[waypointForLabel] = waypointForService
	gw := Gateway{
		APIVersion: "gateway.networking.k8s.io/v1",
		Kind:       "Gateway",
		Metadata: ObjectMeta{
			Name:      waypointName,
			Namespace: stackName,
			Labels:    labels,
		},
		Spec: GatewaySpec{
			GatewayClassName: waypointGatewayClassName,
			Listeners: []GatewayListener{
				{Name: waypointListenerName, Port: waypointListenerPort, Protocol: waypointListenerProto},
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/waypoint.yaml", stackName), gw)
}

// emitBackupEgress compiles one declared external's egress triple: the
// ServiceEntry naming its resolved host/port, and the AuthorizationPolicy
// scoping access to exactly the consuming components' CNPG-created
// ServiceAccounts — the declared backup wiring is the allow-list, other
// identities are denied at this endpoint (plan slice 1) because the
// AuthorizationPolicy's Rules carry only these principals, nothing wider
// (golden rule 53).
func emitBackupEgress(files map[string][]byte, stackName string, t backupEgressTarget) error {
	seLabels := ownershipLabels(stackName, "")
	seLabels[useWaypointLabel] = waypointName
	se := ServiceEntry{
		APIVersion: "networking.istio.io/v1",
		Kind:       "ServiceEntry",
		Metadata: ObjectMeta{
			Name:      t.ExternalName,
			Namespace: stackName,
			Labels:    seLabels,
		},
		Spec: ServiceEntrySpec{
			Hosts:    []string{t.Host},
			Location: "MESH_EXTERNAL",
			Ports: []ServiceEntryPort{
				{Number: t.Port, Name: "backup", Protocol: "TCP"},
			},
			// The host is known exactly from the declared external — no
			// wildcard, so plain per-lookup DNS resolution (not
			// DYNAMIC_DNS, which exists specifically for wildcard hosts)
			// is the honest, simplest choice here.
			Resolution: "DNS",
		},
	}
	if err := set(files, fmt.Sprintf("apps/%s/%s-serviceentry.yaml", stackName, t.ExternalName), se); err != nil {
		return err
	}

	principals := make([]string, 0, len(t.Consumers))
	for _, consumer := range t.Consumers {
		principals = append(principals, principal(stackName, domain.AllowedConsumer{ServiceAccount: consumer}))
	}
	authz := AuthorizationPolicy{
		APIVersion: "security.istio.io/v1",
		Kind:       "AuthorizationPolicy",
		Metadata: ObjectMeta{
			Name:      t.ExternalName + "-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: AuthorizationPolicySpec{
			TargetRefs: []AuthorizationPolicyTargetRef{
				{Group: serviceEntryTargetRefGroup, Kind: serviceEntryTargetRefKind, Name: t.ExternalName},
			},
			Rules: []AuthorizationPolicyRule{
				{From: []AuthorizationPolicyFrom{{Source: AuthorizationPolicySource{Principals: principals}}}},
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-egress-authorizationpolicy.yaml", stackName, t.ExternalName), authz)
}

// emitNeonEgress compiles a managed component's egress triple: an
// EXACT-host ServiceEntry naming pg.EndpointHost (week-four plan,
// 2026-07-27 finding → Revision 2 — see neonPostgresPort's doc comment
// for why this superseded the original domain-pattern design), and the
// AuthorizationPolicy scoping access to exactly its declared
// allowedConsumers. Only ever called for a pg with len(AllowedConsumers)
// > 0 (neonEgressTargets), and domain validation refuses that
// combination whenever EndpointHost is empty (internal/domain/
// postgres.go) — so the empty check below is a defect guard, not a
// reachable compile-time branch under a correctly validated Stack.
func emitNeonEgress(files map[string][]byte, stackName string, pg domain.Postgres) error {
	if pg.EndpointHost == "" {
		return fmt.Errorf(
			"flux emitter: managed component %q declares allowedConsumers with no endpointHost — this is a defect (domain validation should have caught it)",
			pg.Name)
	}

	seName := pg.Name + "-neon"

	seLabels := ownershipLabels(stackName, pg.Name)
	seLabels[useWaypointLabel] = waypointName
	se := ServiceEntry{
		APIVersion: "networking.istio.io/v1",
		Kind:       "ServiceEntry",
		Metadata: ObjectMeta{
			Name:      seName,
			Namespace: stackName,
			Labels:    seLabels,
		},
		Spec: ServiceEntrySpec{
			Hosts:    []string{pg.EndpointHost},
			Location: "MESH_EXTERNAL",
			Ports: []ServiceEntryPort{
				{Number: neonPostgresPort, Name: "postgres-tls", Protocol: "TLS"},
			},
			// The host is now the operator-pinned exact endpoint (never
			// a wildcard — see neonPostgresPort's doc comment), so plain
			// per-lookup DNS resolution is the honest, simplest choice —
			// the same shape neonControlPlaneHost's own entry already
			// proved live.
			Resolution: "DNS",
		},
	}
	if err := set(files, fmt.Sprintf("apps/%s/%s-serviceentry.yaml", stackName, seName), se); err != nil {
		return err
	}

	principals := make([]string, 0, len(pg.AllowedConsumers))
	for _, consumer := range pg.AllowedConsumers {
		principals = append(principals, principal(stackName, consumer))
	}
	authz := AuthorizationPolicy{
		APIVersion: "security.istio.io/v1",
		Kind:       "AuthorizationPolicy",
		Metadata: ObjectMeta{
			Name:      seName + "-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, pg.Name),
		},
		Spec: AuthorizationPolicySpec{
			TargetRefs: []AuthorizationPolicyTargetRef{
				{Group: serviceEntryTargetRefGroup, Kind: serviceEntryTargetRefKind, Name: seName},
			},
			Rules: []AuthorizationPolicyRule{
				{From: []AuthorizationPolicyFrom{{Source: AuthorizationPolicySource{Principals: principals}}}},
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-egress-authorizationpolicy.yaml", stackName, seName), authz)
}

// emitNeonControlPlaneEgress compiles the provisioner's own edge: the
// tf-runner ServiceAccount (emitManagedRunnerRBAC, terraform.go) reaching
// Neon's fixed API host on port 443 — the live-caught bug documented at
// neonControlPlaneHost's doc comment. Scoped to EXACTLY the tf-runner
// identity: a declared consumer (emitNeonEgress's allow-list) has no
// business calling Neon's provisioning API, and tf-runner has no business
// on the data-plane port — the two AuthorizationPolicies never share a
// principal or a port, each own ServiceEntry/policy pair enforcing only
// its own edge (golden rule 53). Called once per stack whenever any
// managed component exists (regardless of declared consumers), the same
// "one shared object, not one per component" discipline emitWaypoint and
// emitManagedRunnerRBAC already follow, since the identity it authorizes
// (tf-runner) is itself shared across every managed component in the
// stack.
func emitNeonControlPlaneEgress(files map[string][]byte, stackName string) error {
	seLabels := ownershipLabels(stackName, "")
	seLabels[useWaypointLabel] = waypointName
	se := ServiceEntry{
		APIVersion: "networking.istio.io/v1",
		Kind:       "ServiceEntry",
		Metadata: ObjectMeta{
			Name:      neonControlPlaneName,
			Namespace: stackName,
			Labels:    seLabels,
		},
		Spec: ServiceEntrySpec{
			Hosts:    []string{neonControlPlaneHost},
			Location: "MESH_EXTERNAL",
			Ports: []ServiceEntryPort{
				{Number: neonControlPlanePort, Name: "https", Protocol: "TLS"},
			},
			// The control-plane host is a fixed, known constant (never a
			// wildcard), so plain per-lookup DNS resolution is the
			// honest, simplest choice — DYNAMIC_DNS exists specifically
			// for wildcard hosts this SE does not declare (the same
			// resolution the pinned data-plane ServiceEntry now uses
			// too — see emitNeonEgress).
			Resolution: "DNS",
		},
	}
	if err := set(files, fmt.Sprintf("apps/%s/%s-serviceentry.yaml", stackName, neonControlPlaneName), se); err != nil {
		return err
	}

	authz := AuthorizationPolicy{
		APIVersion: "security.istio.io/v1",
		Kind:       "AuthorizationPolicy",
		Metadata: ObjectMeta{
			Name:      neonControlPlaneName + "-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: AuthorizationPolicySpec{
			TargetRefs: []AuthorizationPolicyTargetRef{
				{Group: serviceEntryTargetRefGroup, Kind: serviceEntryTargetRefKind, Name: neonControlPlaneName},
			},
			Rules: []AuthorizationPolicyRule{
				{From: []AuthorizationPolicyFrom{{Source: AuthorizationPolicySource{
					Principals: []string{principal(stackName, domain.AllowedConsumer{ServiceAccount: tfRunnerServiceAccountName})},
				}}}},
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-egress-authorizationpolicy.yaml", stackName, neonControlPlaneName), authz)
}

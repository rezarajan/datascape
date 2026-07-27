package flux

import (
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

	// neonWildcardHost and neonPostgresPort are the honest compile-time
	// limit of the managed/Neon egress case (design question the plan
	// flagged explicitly): the Neon endpoint host a managed component's
	// Terraform CR provisions is NOT known at compile time — it is
	// written to the component's credentials Secret only after
	// tofu-controller reconciles (terraform.go's
	// TerraformWriteOutputsToSecret). d7s never reads that Secret (rule
	// 51), so it cannot pin the exact host into a ServiceEntry. What IS
	// knowable at compile time, honestly, is the provider's own domain:
	// every Neon endpoint is a subdomain of neon.tech
	// (ep-<name>.<region>.aws.neon.tech, verified against Neon's own
	// connection docs, 2026-07-26). Compiling a domain-pattern
	// ServiceEntry — never a fabricated specific host — is what
	// "declare + deny" can honestly mean here: the compiled allow-list
	// still enforces identity (only the declared consumer principals may
	// reach anything under this domain through the waypoint), but its
	// SCOPE is the provider's whole domain rather than the one
	// provisioned endpoint. That is a disclosed precision limit, not a
	// best-effort security tier (rule 50) — the enforcement itself
	// (default-deny, identity-scoped allow) is not weakened, only its
	// destination granularity, and only because the exact destination is
	// genuinely runtime-bound information no compile-time artifact can
	// honestly claim to know. Resolution: DYNAMIC_DNS is required for a
	// wildcard host to route at all in ambient mode (verified against
	// istio.io/latest/blog/2026/egress-dynamic-dns/, 2026-07-26) — Envoy
	// resolves each connection's actual backend from the TLS SNI it
	// presents, which is why this ServiceEntry's port is typed TLS
	// rather than opaque TCP. Open item for the live harness (slice 3,
	// not this slice): whether a libpq client's post-SSLRequest TLS
	// handshake presents SNI early enough for Envoy's dynamic
	// forward proxy to route on it is an empirical question this slice
	// does not verify — flagged here rather than assumed.
	neonWildcardHost = "*.neon.tech"
	neonPostgresPort = 5432
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
func backupEgressTargets(selfHosted []domain.Postgres, externals map[string]domain.External) ([]backupEgressTarget, error) {
	byName := make(map[string]*backupEgressTarget)
	var order []string
	for _, pg := range selfHosted {
		if pg.Guarantees.RPO == nil {
			continue
		}
		name := pg.Guarantees.RPO.BackupTo
		t, seen := byName[name]
		if !seen {
			ext, ok := externals[name]
			if !ok {
				return nil, fmt.Errorf(
					"flux emitter: postgres component %q: guarantees.rpo.backupTo %q does not resolve to a declared external — this is a defect (domain validation should have caught it)",
					pg.Name, name)
			}
			host, port, err := objectStoreHostPort(ext)
			if err != nil {
				return nil, err
			}
			t = &backupEgressTarget{ExternalName: name, Host: host, Port: port}
			byName[name] = t
			order = append(order, name)
		}
		t.Consumers = append(t.Consumers, pg.Name)
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
// declares allowedConsumers — the ones egress compilation now covers
// (week-four plan, slice 2's un-refusal). A managed component with no
// declared consumers stays reachable by nothing (the correct
// default-deny state, mirroring AllowedConsumer's own doc comment) and
// compiles no egress objects at all — presence is still the only signal
// (golden rule 50).
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
// that crosses the mesh boundary; otherwise exactly one shared waypoint
// plus one ServiceEntry/AuthorizationPolicy pair per declared external
// and per managed component with declared consumers.
func emitEgress(files map[string][]byte, stackName string, selfHosted, managed []domain.Postgres, externalsByName map[string]domain.External) error {
	backupTargets, err := backupEgressTargets(selfHosted, externalsByName)
	if err != nil {
		return err
	}
	neonTargets := neonEgressTargets(managed)

	if len(backupTargets) == 0 && len(neonTargets) == 0 {
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

// emitNeonEgress compiles a managed component's egress triple: the
// domain-pattern ServiceEntry documented at neonWildcardHost's doc
// comment, and the AuthorizationPolicy scoping access to exactly its
// declared allowedConsumers — the un-refusal this slice ships (week-four
// plan, slice 2).
func emitNeonEgress(files map[string][]byte, stackName string, pg domain.Postgres) error {
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
			Hosts:    []string{neonWildcardHost},
			Location: "MESH_EXTERNAL",
			Ports: []ServiceEntryPort{
				{Number: neonPostgresPort, Name: "postgres-tls", Protocol: "TLS"},
			},
			Resolution: "DYNAMIC_DNS",
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

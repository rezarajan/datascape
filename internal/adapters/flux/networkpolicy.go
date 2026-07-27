package flux

import (
	"fmt"

	"github.com/rezarajan/datascape/internal/domain"
)

// This file compiles the egress DENY FLOOR (week-four plan, Enforcement
// round → Revision 3, "Mechanism correction," 2026-07-27) as Kubernetes
// NetworkPolicy objects — the literal reading of problem definition
// Amendment 2, B2 ("Egress is compiled default-deny") that the prior
// design in egress.go (ServiceEntry + waypoint-attached
// AuthorizationPolicy) turned out NOT to provide on its own.
//
// THE REGISTRY_ONLY DEAD END (dated finding, 2026-07-27): Revision 3's
// first synthesis reached for Istio's mesh-wide outboundTrafficPolicy:
// REGISTRY_ONLY as the enforceable floor — mesh-enrolled workloads
// restricted to registered destinations (cluster-local services plus
// compiled ServiceEntries), leaving identity enforcement to the existing
// compiled AuthorizationPolicy. This does not work: maintainer-confirmed
// unimplemented in ambient (istio discussion #53021), and live-proven
// here — a mesh-enrolled pod completed a full HTTPS round trip to an
// unregistered host (example.com) with REGISTRY_ONLY verified present in
// istiod's config, i.e. the setting is accepted and reported but not
// enforced. It is also, per Istio's own security docs, non-enforcing
// best-effort even in sidecar mode — not a foundation for a compiled
// security guarantee regardless (golden rule 50: no best-effort tier).
//
// THE TWO-LAYER PATTERN'S PROVENANCE: upstream's own blessed answer,
// confirmed via Istio's ambient-mode documentation
// (istio.io/latest/docs/ambient/usage/networkpolicy/, fetched
// 2026-07-27) is that Istio explicitly defers the L3/L4 enforceable-deny
// floor to Kubernetes NetworkPolicy — "Istio respects but doesn't manage
// NetworkPolicy—ambient mode will never bypass it" — while Istio's own
// compiled AuthorizationPolicy/mTLS layer stays authoritative for
// identity (golden rule 50: "Identity is authoritative; network
// segmentation is defense-in-depth"). So this file's NetworkPolicy
// objects are the OUTER, coarser layer (can this pod's traffic leave the
// namespace at all, and to which fixed classes of destination); the
// existing ServiceEntry/AuthorizationPolicy layer (egress.go) remains
// the INNER, identity-scoped layer for the specific declared external
// and managed endpoints. Neither layer alone was sufficient — the
// negative-probe failure that motivated Revision 3 (an undeclared
// identity's SQL reached Neon) is exactly the gap an inner-only design
// leaves: traffic that never touches a compiled ServiceEntry is simply
// unregistered, ordinary egress, invisible to the inner layer entirely.
//
// RULE-58 POSTURE: this slice compiles the floor; it does not prove it
// live (that is explicitly this round's harness-implementer's job, next
// round). Until a live probe from the real consumer's vantage point
// observes both the ALLOW paths still working and a DENY path actually
// refused, this is compiled-but-unverified, not a proven guarantee
// (rule 49: security is proven by the negative probe, never assumed;
// rule 58: a check not run is not a result). The stated environment
// prerequisite (NetworkPolicy-enforcing CNI — kind >= v0.24 natively;
// generic clusters need Calico/Cilium in standard modes, with known
// eBPF-mode eviction/probe caveats) is likewise unverified by this
// slice, and is recorded as such rather than assumed to hold.
const (
	networkPolicyAPIVersion = "networking.k8s.io/v1"

	// kubeSystemNamespace and istioSystemNamespace are matched via
	// Kubernetes' own automatic, immutable namespace label
	// kubernetes.io/metadata.name (stable since Kubernetes 1.21,
	// verified 2026-07-27) — never a custom label this compiler would
	// have to assume the target namespace already carries.
	kubeSystemNamespace  = "kube-system"
	istioSystemNamespace = "istio-system"

	// namespaceNameLabel is the automatic namespace-identity label every
	// Kubernetes namespace carries.
	namespaceNameLabel = "kubernetes.io/metadata.name"

	// dnsPort is kube-dns/CoreDNS's fixed port, both protocols — DNS
	// resolution is cluster-internal, needed regardless of guarantees
	// declared, so it belongs on the shared allow-floor.
	dnsPort = 53

	// istiodXDSPort is istiod's mTLS-secured discovery (xDS) and CA
	// port every mesh-enrolled workload's istio-agent/ztunnel connects
	// to for configuration and certificates (verified against Istio's
	// component-port documentation, 2026-07-27 — 15012 is the TLS/mTLS
	// xDS+CA port; 15010 is its plaintext-only sibling, not used here).
	istiodXDSPort = 15012

	// hboneport is ambient's own secure L4 overlay port: ztunnel tunnels
	// ALL mesh-redirected traffic (in-mesh service calls, waypoint
	// hops, egress through a waypoint) over this port, and NetworkPolicy
	// enforcement — being a host-level, pre-decapsulation mechanism —
	// sees the HBONE wire frame, not the traffic's original destination
	// port (verified verbatim against istio.io/latest/docs/ambient/
	// usage/networkpolicy/, 2026-07-27: "Ambient's secure L4 overlay
	// tunnels traffic between pods over port 15008 ... NetworkPolicy is
	// enforced at the host level (outside pods)... existing policies
	// denying all but specific ports will block HBONE traffic"). That
	// page's own worked examples are INGRESS exceptions; this repo's
	// deny floor is EGRESS, so the same exception is added on the
	// egress side by the same stated mechanism (rule applies to the
	// wire protocol, not the tunneled payload) — deliberately with NO
	// peer restriction (allowed to any destination on this port alone),
	// which is a considered choice, not an oversight: HBONE is itself
	// an authenticated, encrypted, Istio-mTLS tunnel end to end (only a
	// genuine ztunnel/waypoint peer can complete its handshake), so
	// broadening its own transport port doesn't widen who can reach
	// application data — identity enforcement stays with the compiled
	// AuthorizationPolicy layer (golden rule 50), and NetworkPolicy's
	// job here is only to not sever the mesh's own machinery. Scoping
	// this to a specific peer would additionally require the
	// destination NODE's IP range (ztunnel-to-ztunnel hops cross nodes),
	// an environment binding (the node/pod CIDR) no compiled artifact
	// can portably know (golden rules 22, 45) — see the kube API server
	// finding below for the same class of constraint, decided the same
	// way.
	hbonePort = 15008

	// apiServerPort and apiServerServicePort are the two TCP ports the
	// control-plane edge below allows (week-four plan, Control-plane-edge
	// round, 2026-07-27 → Revision 4): 6443, the kube-apiserver's own
	// secure port on most clusters (kind, kubeadm, EKS, GKE all default
	// here), and 443, the port the in-cluster kubernetes.default.svc
	// Service listens on (client-go's in-cluster config, which both
	// CNPG's instance manager and tofu-controller's runner use, reaches
	// the apiserver through that Service rather than the node-level port
	// directly on some clusters). Both are compiled together because a
	// compiled, portable artifact cannot know at compile time which path
	// a given cluster's apiserver client actually takes (golden rules 22,
	// 45) — the owner's decision accepted "destination-unrestricted on
	// these two ports, for these pods" as the disclosed precision limit
	// rather than adding a new schema field for it (see the dated
	// addendum to the KNOWN, DISCLOSED GAP comment below).
	apiServerPort        = 6443
	apiServerServicePort = 443

	// tfRunnerPodSelectorLabel/Value name the pod label the managed-
	// placement control-plane edge below selects by. Verified against
	// tofu-controller's pinned v0.16.4 tag (controllers/
	// tf_controller_runner.go, function runnerPodTemplate, 2026-07-27):
	// the runner Pod's ObjectMeta.Labels set
	// "app.kubernetes.io/name": "tf-runner" as a STATIC, fixed string on
	// every runner pod tofu-controller creates, in every namespace,
	// regardless of the owning Terraform CR's own name — unlike
	// "app.kubernetes.io/instance" (derived from the git revision SHA,
	// e.g. "tf-runner-<8charsha>" — not stable or knowable at compile
	// time) or infrav1.RunnerLabel (= terraform.Namespace, the stack
	// namespace itself — namespace-scoped only, not
	// component-distinguishing, though harmless here since NetworkPolicy
	// is namespace-scoped by default anyway). Both alternatives were
	// considered and rejected for the reasons above;
	// "app.kubernetes.io/name" is the only one of the three that is both
	// reliable and knowable at compile time. There is no pod-level
	// ServiceAccount selector in Kubernetes NetworkPolicy, so this floor
	// cannot reuse tfRunnerServiceAccountName (terraform.go) the way the
	// mesh-layer AuthorizationPolicy precedent (egress.go's
	// emitNeonControlPlaneEgress) scopes by SPIFFE principal instead.
	tfRunnerPodSelectorLabel = "app.kubernetes.io/name"
	tfRunnerPodSelectorValue = "tf-runner"
)

// KNOWN, DISCLOSED GAP — kube API server egress (2026-07-27, this
// slice): CloudNativePG's instance manager (running inside every
// self-hosted Postgres pod, under the Cluster's own ServiceAccount —
// verified against CloudNativePG's own docs, 2026-07-27) calls the
// Kubernetes API server directly (watching its own Cluster resource,
// reading/updating Backup resources) — a real, live dependency, not a
// hypothetical one. tofu-controller's tf-runner pod (managed placement)
// has the same class of need. This deny floor does NOT compile an
// allowance for it, and that omission is deliberate, not silent:
//
//   - The API server's address is not a NetworkPolicy podSelector/
//     namespaceSelector target on most real clusters at all — many
//     managed offerings run it entirely outside the cluster's own pod
//     network (an external endpoint), and even where it looks like a
//     ClusterIP Service, ONLY an ipBlock peer can address it.
//   - That ClusterIP (or, for some installations, the API server's real
//     endpoint IP) is a per-cluster environment binding — a kind
//     cluster, EKS, GKE, and a bare-metal kubeadm cluster do not share
//     one, and are not knowable at compile time. Baking a guessed CIDR
//     into compiled output would both be dishonest (faked precision,
//     exactly what this round's design brief asked NOT to do) and break
//     determinism across environments (golden rules 22, 45).
//   - Many NetworkPolicy-enforcing CNIs document an explicit, automatic
//     exemption for kube-apiserver traffic (since blocking it is almost
//     universally catastrophic) — which may mean this gap never bites
//     on some CNIs and does on others. This slice does not verify
//     either way (rule 58): it is the next round's live harness finding
//     to make, on the actual target CNI, not this slice's to assume.
//
// This is deliberately NOT a compile-refusing declaration: refusing the
// entire deny floor over a secondary, possibly-already-exempted control
// path would be a disproportionate response to an unverified risk, and
// would leave the (verified, upstream-blessed) deny floor itself
// unshipped. Nor does this slice add a new declared escape-hatch field
// (e.g. a compiled ipBlock allowlist knob) — that is meaningfully more
// schema surface than this round dispatched, and is recorded here as a
// FINDING for a follow-up slice if the harness round's live probe shows
// the gap actually bites. If it does, the remedy is either (a) the
// target CNI's own kube-apiserver exemption already covers it, needing
// no compiled change, or (b) a genuinely new declared field naming the
// cluster's own API server CIDR — a scope decision for the owner, not
// this implementer.
//
// SUPERSEDED IN PART (dated addendum, 2026-07-27, week-four plan
// Revision 4, "Control-plane-edge round"): the FINDING above was
// confirmed live — the harness round's own acceptance run hit exactly
// this gap (CI runs 30254082510/30270332686 plus two local
// reproductions, TASK_PROGRESS 2026-07-27): CNPG's initdb bootstrap pod
// failed (exit 1) because it could not reach kube-apiserver, a
// host-network endpoint no NetworkPolicy pod/namespace selector can
// name, and no floor rule allowed it. Owner decision, verbatim
// (2026-07-27, steward question round): "Compile pod+port-scoped edge
// (Recommended)" — reached over (a) a new schema-knob apiserver-endpoint
// field (rejected as more schema surface than warranted now; available
// later as tightening) and (b) reliance on a CNI's own kube-apiserver
// exemption (rejected — refuted by evidence: the mesh dataplane enforces
// the floor regardless of CNI, and assuming an exemption holds would be
// exactly the best-effort tier golden rules 37/50 refuse).
//
// This general-case reasoning above (no compiled apiserver CIDR knob
// exists, still true) is NOT fully superseded: what changed is that the
// floor now DOES compile a narrower, disclosed allowance — not the
// general apiserver-address problem, which remains open. The compiled
// edge (emitCNPGControlPlaneEgress, emitManagedControlPlaneEgress,
// below) covers exactly two pod populations that are both operator-
// managed and implied by declared placement itself: CNPG's own instance
// pods (self-hosted) and tofu-controller's tf-runner pods (managed,
// extending emitNeonControlPlaneEgress's already-recognized provisioner
// edge, egress.go, down to this floor layer). Its own disclosed
// precision limit: destination-UNRESTRICTED on TCP 6443 and 443 for
// those specific pods (apiServerPort/apiServerServicePort's doc comment
// above) — not a pin to the cluster's actual apiserver address, since no
// portable, compile-time-knowable CIDR exists (the general-case gap
// above, unchanged). Any OTHER pod in the namespace remains subject to
// the full deny floor with no apiserver egress at all.

// NetworkPolicy is a Kubernetes networking.k8s.io NetworkPolicy.
type NetworkPolicy struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   ObjectMeta        `yaml:"metadata"`
	Spec       NetworkPolicySpec `yaml:"spec"`
}

// NetworkPolicySpec is the subset of NetworkPolicy.spec d7s emits.
// Egress is deliberately omitted (nil), not an empty slice, when a
// policy declares no rules — the documented Kubernetes shape for "deny
// all egress from the selected pods" once PolicyTypes names Egress;
// yaml.v3's omitempty then drops the key entirely rather than emitting
// a misleading `egress: []`.
type NetworkPolicySpec struct {
	PodSelector NetworkPolicyLabelSelector `yaml:"podSelector"`
	PolicyTypes []string                   `yaml:"policyTypes"`
	Egress      []NetworkPolicyEgressRule  `yaml:"egress,omitempty"`
}

// NetworkPolicyLabelSelector mirrors Kubernetes' standard label-selector
// shape (matchLabels only — this compiler never needs matchExpressions).
// An empty value (MatchLabels nil) marshals as `{}`, Kubernetes' own
// "match everything this selector's context allows" convention — used
// both for "every pod in this namespace" (NetworkPolicySpec.PodSelector)
// and "every pod in the peer's target namespace"
// (NetworkPolicyPeer.PodSelector).
type NetworkPolicyLabelSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels,omitempty"`
}

// NetworkPolicyEgressRule is one entry of NetworkPolicySpec.egress. Both
// To and Ports are nil-omittable independently: a rule with neither
// (NetworkPolicyEgressRule{}) allows ALL egress traffic — Kubernetes'
// own "no restriction" shape, used only for the waypoint's own grant
// (emitWaypointNetworkPolicy).
type NetworkPolicyEgressRule struct {
	To    []NetworkPolicyPeer `yaml:"to,omitempty"`
	Ports []NetworkPolicyPort `yaml:"ports,omitempty"`
}

// NetworkPolicyPeer is one entry of an egress rule's to. NamespaceSelector
// and PodSelector are pointers specifically so a present-but-empty
// selector (`&NetworkPolicyLabelSelector{}`, marshaling as `{}`) can be
// distinguished from an absent one (nil, omitted) — Kubernetes gives
// these two shapes different meanings (PodSelector alone, no
// NamespaceSelector, means "in the policy's own namespace"; both
// present/empty broadens to "every pod in the matched namespace(s)").
type NetworkPolicyPeer struct {
	NamespaceSelector *NetworkPolicyLabelSelector `yaml:"namespaceSelector,omitempty"`
	PodSelector       *NetworkPolicyLabelSelector `yaml:"podSelector,omitempty"`
}

// NetworkPolicyPort is one entry of an egress rule's ports.
type NetworkPolicyPort struct {
	Protocol string `yaml:"protocol"`
	Port     int    `yaml:"port"`
}

// emitNetworkPolicies compiles the deny floor for stackName (week-four
// plan, Enforcement round → Revision 3, extended by the Control-plane-
// edge round → Revision 4): default-deny-egress (a), allow-cluster-egress
// (b: DNS, istiod, same-namespace, HBONE), — only when a waypoint
// actually exists in this stack (waypointPresent, threaded from
// emitEgress's own return so this file never references a pod selector
// matching nothing, golden rule 24's "no dangling edge" spirit) —
// allow-waypoint-egress (c): the mesh's sole external gate, left
// otherwise unrestricted because it is already guarded by the compiled
// ServiceEntry/AuthorizationPolicy allow-list (egress.go), the
// authoritative identity layer this NetworkPolicy floor is deliberately
// not duplicating (golden rule 50) — and, per Revision 4, one
// control-plane-edge object per self-hosted component plus (once per
// stack) one shared control-plane-edge object for every managed
// component's tf-runner pods (see the KNOWN, DISCLOSED GAP comment's
// 2026-07-27 addendum above, and emitCNPGControlPlaneEgress /
// emitManagedControlPlaneEgress below).
func emitNetworkPolicies(files map[string][]byte, stackName string, waypointPresent bool, selfHosted, managed []domain.Postgres) error {
	if err := emitDefaultDenyEgress(files, stackName); err != nil {
		return err
	}
	if err := emitAllowClusterEgress(files, stackName); err != nil {
		return err
	}
	if waypointPresent {
		if err := emitAllowWaypointEgress(files, stackName); err != nil {
			return err
		}
	}
	for _, pg := range selfHosted {
		if err := emitCNPGControlPlaneEgress(files, stackName, pg); err != nil {
			return err
		}
	}
	if len(managed) > 0 {
		if err := emitManagedControlPlaneEgress(files, stackName); err != nil {
			return err
		}
	}
	return nil
}

// emitDefaultDenyEgress compiles the floor itself (bullet a): every pod
// in the namespace, no egress rules at all — Amendment 2's "compiled
// default-deny" made literal, per this round's Mechanism correction.
func emitDefaultDenyEgress(files map[string][]byte, stackName string) error {
	np := NetworkPolicy{
		APIVersion: networkPolicyAPIVersion,
		Kind:       "NetworkPolicy",
		Metadata: ObjectMeta{
			Name:      "default-deny-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: NetworkPolicySpec{
			PodSelector: NetworkPolicyLabelSelector{},
			PolicyTypes: []string{"Egress"},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/networkpolicy-default-deny-egress.yaml", stackName), np)
}

// emitAllowClusterEgress compiles bullet (b): the fixed allowlist every
// mesh-enrolled workload needs regardless of which guarantees it
// declares — DNS, the istiod control plane, same-namespace peers, and
// the HBONE tunnel port (see hbonePort's doc comment for why that last
// one carries no peer restriction). Additive with
// emitDefaultDenyEgress's policy (both select every pod in the
// namespace; Kubernetes unions egress rules across every NetworkPolicy
// that selects a given pod for Egress), so this is a second object, not
// a merge into the first — keeping the "floor" and "the fixed
// allowance" auditable as separate, individually reviewable compiled
// facts.
func emitAllowClusterEgress(files map[string][]byte, stackName string) error {
	np := NetworkPolicy{
		APIVersion: networkPolicyAPIVersion,
		Kind:       "NetworkPolicy",
		Metadata: ObjectMeta{
			Name:      "allow-cluster-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: NetworkPolicySpec{
			PodSelector: NetworkPolicyLabelSelector{},
			PolicyTypes: []string{"Egress"},
			Egress: []NetworkPolicyEgressRule{
				// kube-dns / CoreDNS.
				{
					To: []NetworkPolicyPeer{
						{NamespaceSelector: &NetworkPolicyLabelSelector{
							MatchLabels: map[string]string{namespaceNameLabel: kubeSystemNamespace},
						}},
					},
					Ports: []NetworkPolicyPort{
						{Protocol: "UDP", Port: dnsPort},
						{Protocol: "TCP", Port: dnsPort},
					},
				},
				// istiod control plane (xDS + CA, mTLS).
				{
					To: []NetworkPolicyPeer{
						{NamespaceSelector: &NetworkPolicyLabelSelector{
							MatchLabels: map[string]string{namespaceNameLabel: istioSystemNamespace},
						}},
					},
					Ports: []NetworkPolicyPort{
						{Protocol: "TCP", Port: istiodXDSPort},
					},
				},
				// Same-namespace pods (e.g. CNPG's own pods, a declared
				// consumer colocated with the component it reaches) — no
				// port restriction, since fine-grained access within the
				// namespace is the compiled AuthorizationPolicy's job
				// (golden rule 50), not this floor's.
				{
					To: []NetworkPolicyPeer{
						{PodSelector: &NetworkPolicyLabelSelector{}},
					},
				},
				// HBONE — ambient's own tunnel wire protocol (see
				// hbonePort's doc comment for why no peer restriction).
				{
					Ports: []NetworkPolicyPort{
						{Protocol: "TCP", Port: hbonePort},
					},
				},
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/networkpolicy-allow-cluster-egress.yaml", stackName), np)
}

// emitAllowWaypointEgress compiles bullet (c): the waypoint pod itself
// (matched by istio.io/gateway-name, the label istiod's gateway-
// deployment-controller applies to every waypoint pod it provisions,
// named after the owning Gateway resource — verified 2026-07-27) is the
// sole external gate, so it alone is granted unrestricted egress: an
// empty NetworkPolicyEgressRule (neither To nor Ports set) is
// Kubernetes' own "allow everything" shape. This does not widen who can
// reach through it — the waypoint only ever forwards traffic the
// compiled ServiceEntry/AuthorizationPolicy pair already authorized
// (egress.go); this object exists purely so the waypoint's own
// machinery (which itself must dial the real external endpoint on the
// real internet, an address this floor cannot enumerate) is not cut off
// by the same default-deny baseline every other pod in the namespace
// gets.
func emitAllowWaypointEgress(files map[string][]byte, stackName string) error {
	np := NetworkPolicy{
		APIVersion: networkPolicyAPIVersion,
		Kind:       "NetworkPolicy",
		Metadata: ObjectMeta{
			Name:      "allow-waypoint-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: NetworkPolicySpec{
			PodSelector: NetworkPolicyLabelSelector{
				MatchLabels: map[string]string{"istio.io/gateway-name": waypointName},
			},
			PolicyTypes: []string{"Egress"},
			Egress:      []NetworkPolicyEgressRule{{}},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/networkpolicy-allow-waypoint-egress.yaml", stackName), np)
}

// controlPlaneEgressRules is the egress rule the control-plane-edge
// objects below share: TCP 6443 and TCP 443, no `To` peer restriction —
// mirroring the shape of the HBONE rule above (hbonePort's doc comment)
// for the same reason: the two ports name a specific, known destination
// class (the cluster's own kube-apiserver, reachable either directly or
// through the kubernetes.default.svc Service), but that destination is a
// host-network endpoint no NetworkPolicy pod/namespace selector can name
// (see apiServerPort's doc comment and the KNOWN, DISCLOSED GAP
// addendum), so the allowance is scoped by SOURCE pod (the caller of
// emitCNPGControlPlaneEgress/emitManagedControlPlaneEgress) rather than
// by destination.
func controlPlaneEgressRules() []NetworkPolicyEgressRule {
	return []NetworkPolicyEgressRule{
		{
			Ports: []NetworkPolicyPort{
				{Protocol: "TCP", Port: apiServerPort},
				{Protocol: "TCP", Port: apiServerServicePort},
			},
		},
	}
}

// emitCNPGControlPlaneEgress compiles the self-hosted half of Revision
// 4's control-plane edge (week-four plan, Control-plane-edge round,
// 2026-07-27): one NetworkPolicy per self-hosted Postgres component,
// podSelector-scoped to that Cluster's own pods via cnpg.io/cluster —
// the exact same label/value convention flux.go's zero-trust
// AuthorizationPolicy selector already uses for the same Cluster's pods
// (flux.go, emitZeroTrust). CNPG's instance manager (running under the
// Cluster's own ServiceAccount inside every instance pod) calls the
// kube-apiserver directly during initdb bootstrap and ongoing
// reconciliation — an edge implied by placement: self-hosted itself
// (every self-hosted component's Cluster needs it, unconditionally), not
// a new declared field, the same reasoning class as
// emitNeonControlPlaneEgress's provisioner edge (egress.go). Compiled
// once per self-hosted component (unlike the shared, once-per-stack
// managed-side object below) because each component's Cluster pods carry
// their own, component-specific cnpg.io/cluster value — there is no
// single shared label spanning every self-hosted component's pods the
// way tf-runner's fixed label spans every managed component's runner
// pods.
func emitCNPGControlPlaneEgress(files map[string][]byte, stackName string, pg domain.Postgres) error {
	np := NetworkPolicy{
		APIVersion: networkPolicyAPIVersion,
		Kind:       "NetworkPolicy",
		Metadata: ObjectMeta{
			Name:      pg.Name + "-controlplane-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, pg.Name),
		},
		Spec: NetworkPolicySpec{
			PodSelector: NetworkPolicyLabelSelector{
				MatchLabels: map[string]string{"cnpg.io/cluster": pg.Name},
			},
			PolicyTypes: []string{"Egress"},
			Egress:      controlPlaneEgressRules(),
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-networkpolicy-controlplane-egress.yaml", stackName, pg.Name), np)
}

// emitManagedControlPlaneEgress compiles the managed half of Revision
// 4's control-plane edge: ONE shared NetworkPolicy per stack (not one
// per component), podSelector-scoped to tfRunnerPodSelectorLabel/Value
// (see that constant's doc comment for the tofu-controller v0.16.4
// source verification and the two rejected label alternatives). Every
// managed component's Terraform CR runs under a runner pod carrying the
// same static label, so a single shared object covers all of them — the
// same "one shared object, not one per component" discipline
// emitWaypoint, emitManagedRunnerRBAC, and emitNeonControlPlaneEgress
// already follow (egress.go), extending that provisioner edge down to
// this floor layer: tf-runner needs the apiserver both for its own
// Terraform state/status reconciliation and for the same class of
// kube-apiserver dependency CNPG's instance manager has above.
func emitManagedControlPlaneEgress(files map[string][]byte, stackName string) error {
	np := NetworkPolicy{
		APIVersion: networkPolicyAPIVersion,
		Kind:       "NetworkPolicy",
		Metadata: ObjectMeta{
			Name:      "tf-runner-controlplane-egress",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: NetworkPolicySpec{
			PodSelector: NetworkPolicyLabelSelector{
				MatchLabels: map[string]string{tfRunnerPodSelectorLabel: tfRunnerPodSelectorValue},
			},
			PolicyTypes: []string{"Egress"},
			Egress:      controlPlaneEgressRules(),
		},
	}
	return set(files, fmt.Sprintf("apps/%s/networkpolicy-tf-runner-controlplane-egress.yaml", stackName), np)
}

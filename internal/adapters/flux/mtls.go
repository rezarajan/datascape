package flux

// PeerAuthentication is an Istio security.istio.io PeerAuthentication.
type PeerAuthentication struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   ObjectMeta             `yaml:"metadata"`
	Spec       PeerAuthenticationSpec `yaml:"spec"`
}

// PeerAuthenticationSpec is the subset of PeerAuthentication.spec d7s
// emits: a namespace-wide STRICT mTLS mode. d7s's own schema has no
// field to weaken this once the mtls guarantee is declared (golden
// rule 50).
type PeerAuthenticationSpec struct {
	MTLS PeerAuthenticationMTLS `yaml:"mtls"`
}

// PeerAuthenticationMTLS is PeerAuthentication.spec.mtls.
type PeerAuthenticationMTLS struct {
	Mode string `yaml:"mode"`
}

// AuthorizationPolicy is an Istio security.istio.io AuthorizationPolicy.
type AuthorizationPolicy struct {
	APIVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	Metadata   ObjectMeta              `yaml:"metadata"`
	Spec       AuthorizationPolicySpec `yaml:"spec"`
}

// AuthorizationPolicySpec is the subset of AuthorizationPolicy.spec d7s
// emits. An empty Rules list denies all traffic to the selected
// workload — Istio's documented default-deny shape; non-empty Rules
// allow only the principals compiled from declared wiring (golden
// rule 53).
type AuthorizationPolicySpec struct {
	Selector AuthorizationPolicySelector `yaml:"selector"`
	Rules    []AuthorizationPolicyRule   `yaml:"rules"`
}

// AuthorizationPolicySelector is AuthorizationPolicy.spec.selector.
type AuthorizationPolicySelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

// AuthorizationPolicyRule is one entry of AuthorizationPolicy.spec.rules.
// To scopes the rule to specific ports: found necessary while running
// the acceptance harness live — an unscoped allow-list also gates CNPG's
// own operator-to-instance status traffic (port 8000), which isn't a
// declared consumer but an operational necessity of the durability
// guarantee's own emitted infra (golden rule 40: only a real workload on
// real infrastructure proved this).
type AuthorizationPolicyRule struct {
	From []AuthorizationPolicyFrom `yaml:"from"`
	To   []AuthorizationPolicyTo   `yaml:"to,omitempty"`
}

// AuthorizationPolicyFrom is one entry of an AuthorizationPolicyRule.from.
type AuthorizationPolicyFrom struct {
	Source AuthorizationPolicySource `yaml:"source"`
}

// AuthorizationPolicySource names the allowed peer. Principals resolve a
// declared ServiceAccount to its mesh identity — never a hand-constructed
// address (golden rule 15). Namespaces matches any identity within a
// given namespace, used only for the operator's own control-plane rule,
// never for declared consumers (those are always principal-scoped).
type AuthorizationPolicySource struct {
	Principals []string `yaml:"principals,omitempty"`
	Namespaces []string `yaml:"namespaces,omitempty"`
}

// AuthorizationPolicyTo is one entry of an AuthorizationPolicyRule.to.
type AuthorizationPolicyTo struct {
	Operation AuthorizationPolicyOperation `yaml:"operation"`
}

// AuthorizationPolicyOperation scopes a rule to specific destination
// ports.
type AuthorizationPolicyOperation struct {
	Ports []string `yaml:"ports,omitempty"`
}

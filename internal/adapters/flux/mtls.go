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
type AuthorizationPolicyRule struct {
	From []AuthorizationPolicyFrom `yaml:"from"`
}

// AuthorizationPolicyFrom is one entry of an AuthorizationPolicyRule.from.
type AuthorizationPolicyFrom struct {
	Source AuthorizationPolicySource `yaml:"source"`
}

// AuthorizationPolicySource names the allowed peer by its mesh identity
// (SPIFFE principal), resolved from the declared ServiceAccount — never
// a hand-constructed address (golden rule 15).
type AuthorizationPolicySource struct {
	Principals []string `yaml:"principals"`
}

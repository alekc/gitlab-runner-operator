package v1beta2

// EffectiveNamespace returns the namespace the kubernetes executor runs job
// pods in: the explicit Namespace when set, otherwise the supplied fallback
// (the runner object's own namespace). This is the single source of truth for
// the namespace-defaulting rule, shared by config.toml rendering and RBAC
// provisioning so the two cannot drift.
func (k *KubernetesConfig) EffectiveNamespace(fallback string) string {
	if k != nil && k.Namespace != "" {
		return k.Namespace
	}
	return fallback
}

// BuildNamespaceAllowed reports whether the executor may run job pods in ns. An
// empty value or the runner's own namespace is always allowed; otherwise ns must
// be in allowed or allowed must contain "*". Cross-namespace RBAC is a
// privilege-escalation vector unless an operator admin opts in via the flag.
func BuildNamespaceAllowed(ns, ownNamespace string, allowed []string) bool {
	if ns == "" || ns == ownNamespace {
		return true
	}
	for _, a := range allowed {
		if a == "*" || a == ns {
			return true
		}
	}
	return false
}

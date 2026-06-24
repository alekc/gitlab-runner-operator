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

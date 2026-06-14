---
name: k8s-explain
description: "Explain a Kubernetes manifest (Deployment, StatefulSet, Service, Ingress, NetworkPolicy, RBAC, CRD instance) in plain language and flag risks. Use when the user pastes YAML or asks 'what does this manifest do' or 'review this manifest'."
---

When triggered:

1. Identify `kind`, `apiVersion`, and `metadata.namespace` (note if missing).
2. Summarize intent in 2 lines.
3. Walk the spec in this order, listing only fields that are set:
   - **Pod template:** image (tag/digest), command/args, env, ports, resources (requests/limits), probes, securityContext, volume mounts, serviceAccountName.
   - **Workload:** replicas, selector, strategy/updateStrategy.
   - **Service/Ingress:** type, ports, selector, hosts, TLS.
   - **NetworkPolicy:** podSelector, ingress/egress rules.
   - **RBAC:** subjects, verbs, resources, scope (Role vs ClusterRole).
4. Use `kubectl explain <kind>.<field>` (recursive=false) when a field's purpose is non-obvious.
5. Flag risks under a **Risks** heading. Cover at minimum:
   - Missing `resources.requests` or `resources.limits`.
   - Missing readiness or liveness probe.
   - Image pinned to `:latest` or no digest.
   - `runAsRoot`, `privileged`, `hostNetwork`, `hostPath`, `allowPrivilegeEscalation: true`.
   - Broad RBAC (`verbs: ["*"]`, `create`/`patch` on Secret, `*` resource).
   - `replicas > 1` without a PodDisruptionBudget.
   - Service `type: LoadBalancer` without an explicit `loadBalancerSourceRanges`.

## Output format

- **Summary** (2 lines)
- **Spec walk** (bullets, one `field: value` per line)
- **Risks** (bullets, each one cites the specific field)

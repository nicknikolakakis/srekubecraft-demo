---
name: k8s-debug
description: "Diagnose a Kubernetes pod that is crashing, restarting, or stuck (CrashLoopBackOff, ImagePullBackOff, Pending, OOMKilled). Use when the user mentions a pod, restarts, probe failures, or asks 'why is X not running'."
---

When triggered, run this investigation in order. Stop early once the root cause is clear.

1. **Identify target.** Confirm the namespace and pod name (or label selector). If the user says "default", treat it as the `default` namespace explicitly.

2. **Status snapshot.**
   - `kubectl get pod <name> -n <ns> -o wide`
   - `kubectl get pod <name> -n <ns> -o jsonpath='{.status.containerStatuses[*].state}'`

3. **Describe.** `kubectl describe pod <name> -n <ns>`. Read the Events tail and each container's `LastState`.

4. **Logs.** For every container with restarts:
   - Current: `kubectl logs <name> -n <ns> -c <container>`
   - Previous (post-crash): `kubectl logs <name> -n <ns> -c <container> --previous`

5. **Namespace events.** `kubectl get events -n <ns> --sort-by=.lastTimestamp` and look at the last ~30 entries.

6. **Resource pressure.** If `LastState` is `OOMKilled` or the node looks pressured: `kubectl top pod -n <ns>` and `kubectl describe node <node>`.

7. **Probes.** Pull readiness/liveness probe spec from `describe`. Note path, port, timeout, periodSeconds, failureThreshold; cross-reference with probe-failure events.

## Output format

- **Verdict:** one sentence, e.g. `OOMKilled: container memory limit 256Mi, working set ~290Mi.`
- **Evidence:** 3 to 5 bullets citing specific fields and log lines.
- **Suggested fix:** a concrete kubectl command or YAML diff. Do **not** run it; the user will.

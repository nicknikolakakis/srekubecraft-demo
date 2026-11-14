#!/usr/bin/env bash
# usage: timeline.sh <pod> [context] [namespace]
set -euo pipefail

pod="$1"
ctx="${2:-kind-imgvol-demo}"
ns="${3:-models}"
k() { kubectl --context "$ctx" -n "$ns" "$@"; }

echo "== $pod: seconds since creation (API timestamps, 1s precision)"
k get pod "$pod" -o json | jq -r '
  (.metadata.creationTimestamp | fromdateiso8601) as $t0
  | def at(c): ([.status.conditions[] | select(.type == c) | .lastTransitionTime][0] // null);
    def rel(ts): if ts == null then "-" else ((ts | fromdateiso8601) - $t0 | tostring) end;
  [
    ["scheduled", rel(at("PodScheduled"))],
    ["sandbox_ready", rel(at("PodReadyToStartContainers"))],
    ["initialized", rel(at("Initialized"))],
    ["container_started", rel(.status.containerStatuses[0].state.running.startedAt // null)],
    ["ready", rel(at("Ready"))]
  ][] | "  \(.[0])\t\(.[1])"'

echo "== image pulls (kubelet events)"
# by uid, so earlier pods with the same name do not leak in
uid="$(k get pod "$pod" -o jsonpath='{.metadata.uid}')"
k get events --field-selector "involvedObject.uid=$uid,reason=Pulled" -o json | jq -r '
  .items | sort_by(.firstTimestamp) | .[].message
  | capture("image \"(?<img>[^\"]+)\" (in (?<d>[0-9.]+[mµ]?s)|(?<cached>already present))")
  | "  \(.img)\t\(.d // "cached")"'

echo "== runtime"
k logs "$pod" -c vllm 2>/dev/null | grep -oE 'Loading weights took [0-9.]+ seconds|init engine .* took [0-9.]+ s' | sed 's/^/  /' || true

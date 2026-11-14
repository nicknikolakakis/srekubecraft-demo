#!/usr/bin/env bash
# usage: [SIZE=1.5b] run.sh <imagevolume|hf|lan|pvc|populate> <run-id>
set -euo pipefail

method="$1"
id="$2"
size="${SIZE:-0.5b}"
case "$size" in
  0.5b) rev=7ae557604adf67be50417f59c2c2f167def9a775 ;;
  1.5b) rev=989aa7980e4cf806f80c7fef2b1adb7bc71aa306 ;;
  *) echo "unknown SIZE: $size" >&2; exit 1 ;;
esac
# object names cannot hold dots
slug="$(tr -d . <<<"$size")"
# manifests are written for 0.5b, rewrite every model-specific name
sized() {
  [ "$size" = 0.5b ] && { cat; return; }
  sed -e "s/qwen2\.5-0\.5b/qwen2.5-$size/g" \
      -e "s#Qwen2\.5-0\.5B-Instruct/resolve/[0-9a-f]*#Qwen2.5-$(tr b B <<<"$size")-Instruct/resolve/$rev#" \
      -e "s/qwen-weights\$/qwen-weights-$slug/" \
      -e "s#http://fileserver#http://fileserver-$slug#" \
      -e "s/storage: 2Gi/storage: 4Gi/"
}
ctx="kind-imgvol-demo"
ns="models"
dir="$(cd "$(dirname "$0")/.." && pwd)"
w="$dir/kubernetes/workloads"
k() { kubectl --context "$ctx" -n "$ns" "$@"; }

case "$method" in
  imagevolume) src="$w/30-image-volume.yaml"; old="qwen-imagevolume" ;;
  hf)          src="$w/10-init-download.yaml"; old="qwen-download" ;;
  lan)         src="$w/10-init-download.yaml"; old="qwen-download" ;;
  pvc)         src="$w/21-pvc-pod.yaml"; old="qwen-pvc" ;;
  populate)
    # one-off: mirror + PVC + populate Job for this size, no pod to time
    sfx=""
    [ "$size" = 0.5b ] || sfx="-$slug"
    sed "s/name: fileserver\$/name: fileserver$sfx/; s/app.kubernetes.io\/name: fileserver\$/app.kubernetes.io\/name: fileserver$sfx/" "$w/05-fileserver.yaml" | sized | k apply -f -
    k wait --for=condition=Ready "pod/fileserver$sfx" --timeout=300s
    sed "s/name: populate\$/name: populate$sfx/" "$w/20-pvc.yaml" | sized | k apply -f -
    k wait --for=condition=complete "job/populate$sfx" --timeout=600s
    k get job "populate$sfx" -o jsonpath='populate: {.status.startTime} -> {.status.completionTime}{"\n"}'
    exit 0 ;;
  *) echo "unknown method: $method" >&2; exit 1 ;;
esac
pod="qwen-$method-$id-$slug"

manifest="$(sed "s/name: $old\$/name: $pod/" "$src")"
# lan reuses the hf manifest pointed at the in-cluster mirror
[ "$method" = lan ] && manifest="$(sed 's#value: https://huggingface.co/.*#value: http://fileserver#' <<<"$manifest")"
manifest="$(sized <<<"$manifest")"
k apply -f - <<<"$manifest" >/dev/null

for _ in $(seq 1 300); do
  ready="$(k get pod "$pod" -o jsonpath='{.status.containerStatuses[0].ready}' 2>/dev/null || true)"
  restarts="$(k get pod "$pod" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || true)"
  [ "$ready" = true ] && break
  [ "${restarts:-0}" -gt 0 ] && { echo "$pod restarted, aborting" >&2; exit 1; }
  sleep 2
done
[ "$ready" = true ] || { echo "$pod not ready after 600s" >&2; exit 1; }

"$dir/scripts/timeline.sh" "$pod" "$ctx" "$ns"

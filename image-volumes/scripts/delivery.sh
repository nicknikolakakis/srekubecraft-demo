#!/usr/bin/env bash
# usage: [HF=1] delivery.sh <round>   (run crictl rmi on the worker between rounds)
set -euo pipefail

round="$1"
ctx="kind-imgvol-demo"
ns="models"
dir="$(cd "$(dirname "$0")/.." && pwd)"
tpl="$dir/kubernetes/workloads/40-delivery.yaml"
out="$dir/results/delivery.tsv"
k() { kubectl --context "$ctx" -n "$ns" "$@"; }
mkdir -p "$dir/results"
[ -s "$out" ] || printf 'round\tsize\tmethod\tseconds\tnote\n' >"$out"

rev() { case "$1" in 0.5b) echo 7ae557604adf67be50417f59c2c2f167def9a775 ;; 1.5b) echo 989aa7980e4cf806f80c7fef2b1adb7bc71aa306 ;; esac; }

# method: iv-cold | iv-warm | lan | hf
one() {
  local size="$1" method="$2" slug pod url vol
  slug="$(tr -d . <<<"$size")"
  pod="dlv-$method-$slug-r$round"
  url=""
  vol="emptyDir: {sizeLimit: 4Gi}"
  case "$method" in
    iv-*) vol="image: {reference: localhost:5001/weights/qwen2.5-$size:v1, pullPolicy: IfNotPresent}" ;;
    lan) url="http://fileserver$([ "$size" = 0.5b ] || echo "-$slug")" ;;
    hf) url="https://huggingface.co/Qwen/Qwen2.5-$(tr b B <<<"$size")-Instruct/resolve/$(rev "$size")" ;;
  esac
  sed -e "s#__NAME__#$pod#" -e "s#__URL__#$url#" -e "s#__VOLUME__#$vol#" "$tpl" | k apply -f - >/dev/null
  k wait --for=jsonpath='{.status.phase}'=Succeeded "pod/$pod" --timeout=900s >/dev/null

  local secs note=""
  if [ -n "$url" ]; then
    secs="$(k logs "$pod" | sed -n 's/^download_seconds=//p')"
  else
    # kubelet Pulled event for the volume image, e.g. "in 5.492s" or "already present"
    local msg uid
    uid="$(k get pod "$pod" -o jsonpath='{.metadata.uid}')"
    msg="$(k get events --field-selector "involvedObject.uid=$uid,reason=Pulled" -o jsonpath='{range .items[*]}{.message}{"\n"}{end}' | grep "weights/qwen2.5-$size")"
    if grep -q 'already present' <<<"$msg"; then
      secs=0; note=cached
    else
      secs="$(sed -nE 's/.* in ([0-9.]+)(m?s) .*/\1 \2/p' <<<"$msg" | awk '{print ($2=="ms") ? $1/1000 : $1}')"
      note="$(sed -nE 's/.*Image size: ([0-9]+) bytes.*/\1 bytes/p' <<<"$msg")"
    fi
  fi
  printf '%s\t%s\t%s\t%s\t%s\n' "$round" "$size" "$method" "$secs" "$note" | tee -a "$out"
}

for size in 0.5b 1.5b; do
  one "$size" iv-cold
  one "$size" iv-warm
  one "$size" lan
  if [ "${HF:-0}" = 1 ]; then one "$size" hf; fi
done

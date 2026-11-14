# SREKubeCraft Demo

<p align="center">
  <a href="https://srekubecraft.io/">Blog</a>
  &nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="CONTRIBUTING.md">Contributing</a>
  &nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="https://github.com/nicknikolakakis/srekubecraft-demo/issues">Issues</a>
</p>

<p align="center">
  <a href="LICENSE">
    <img alt="License" src="https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge">
  </a>
</p>

---

Companion code for [srekubecraft.io](https://srekubecraft.io/) blog posts. Each directory is a self-contained demo you can run on your own cluster.

## Demos

| Directory | Blog Post | What It Covers |
|-----------|-----------|----------------|
| `knative/` | [Knative - The Platform Engineer's Guide to Serverless on Kubernetes](https://srekubecraft.io/posts/knative/) | Knative Serving & Eventing |
| `oauth2-proxy-mcp/` | [OAuth2-Proxy - Securing MCP Servers on Kubernetes](https://srekubecraft.io/posts/oauth2-proxy/) | OAuth2-Proxy sidecar + DBHub MCP + NetworkPolicies |
| `dapr/` | [Dapr - Building a Safe Platform for Citizen Developer Apps on Kubernetes](https://srekubecraft.io/posts/dapr/) | Dapr runtime + shared Helm chart + Harbor OCI + Flux GitOps + Vault/ESO secrets + node isolation |
| `kserve/` | [KServe - Production ML Serving on Kubernetes, from sklearn to LLMs](https://srekubecraft.io/posts/kserve/) | KServe Serverless mode + Knative + Istio + cert-manager + sklearn-iris (scale-to-zero) + Ollama LLM custom predictor (OpenAI-compatible chat API) |
| `adk-agent/` | [Building a Read-Only Kubernetes Agent with Google ADK (Go)](https://srekubecraft.io/posts/kubernetes-agent-with-google-adk-go/) | Single-binary Go agent on Google ADK + Gemini, packaged as a scratch image, deployed via Helm with three secret modes (inline / existingSecret / ESO). Four-layer defense: prompt, kubectl verb whitelist, read-only RBAC, restricted PodSecurity. |
| `llm-d/` | [llm-d - Kubernetes-Native Distributed LLM Inference at Scale](https://srekubecraft.io/posts/llm-d-distributed-inference/) | llm-d router (Endpoint Picker + Envoy) + Gateway API Inference Extension + `InferencePool` + 4x `llm-d-inference-sim` (GPU-free vLLM simulator). Inference-aware routing (prefix-cache / KV / queue scorers) via Flux on Kind. |
| `modelplane-fleet/` | [Modelplane - Fleet Inference Across Clusters, Demoed Without a Single GPU](https://srekubecraft.io/posts/modelplane-fleet-inference/) | Modelplane v0.3.1 on Crossplane 2.4.0: one control plane + two BYO Kind "regions" with `dra-example-driver` fake GPUs and a mock engine. Fleet placement, single Traefik front door routing cross-cluster, `NoExecute` drain and shortfall. No GPU, no cloud. |
| `gang-scheduling/` | [Kubernetes 1.37 Gang Scheduling - The Workload API, a Silently Dropped Field, and Where Preemption Stops](https://srekubecraft.io/posts/kubernetes-gang-scheduling-workload-api/) | KEP-4671 `Workload`/`PodGroup` APIs (beta in v1.37) on a four-node Kind cluster with `dra-example-driver` fake GPUs. Partial-placement deadlock, `gang.minCount` fixing it, the `WorkloadWithJob` gate that prunes the Job field silently, and the two limits: no DRA preemption, and `disruptionMode` defaulting to `single`. No GPU. |
| `karmada-multi-component/` | [Karmada v1.19 Multi-Component Scheduling - One Training Job, Three Clusters, Four Silent Failures](https://srekubecraft.io/posts/karmada-multi-component-scheduling/) | Karmada v1.19 host + three member Kind clusters with fake `nvidia.com/gpu` capacity and the Kubeflow training-operator v1.9.4. A `PyTorchJob` copied to every member with the gate off and on, the exact spread constraint that enables whole-job placement, GPU-blind default resource models, and the general estimator placing a job where its workers can never start. No GPU. |
| `openshell-redteam/` | [Red-Teaming My Own Agent Sandbox - 15 Exfiltration Techniques vs OpenShell](https://srekubecraft.io/posts/red-teaming-my-agent-sandbox-openshell/) | NVIDIA OpenShell v0.1.5 agent sandbox attacked with 15 exfiltration/escape techniques through `openshell sandbox exec`, no LLM in the loop. Deny-by-default egress, per-path binary identity, synthetic DNS, credential placeholders, Landlock reads, cgroup containment, and an L7 `chat_id` rule that closes the worst gap. Canary credentials only, OCSF evidence per technique. Docker, no Kubernetes. |
| `image-volumes/` | [Image Volumes Went GA in Kubernetes 1.36 - Should You Ship Your Models as OCI Images?](https://srekubecraft.io/posts/kubernetes-image-volumes-model-weights/) | Qwen2.5 0.5B and 1.5B weights as a `FROM scratch` OCI image mounted into vLLM CPU v0.33.0 through an image volume on Kind (K8s 1.37, containerd 2.3.4), timed against a Hugging Face download, an in-cluster HTTP mirror and a pre-populated PVC. Warm image volumes cost 0s, cold ones are 2.5 to 4x slower than a plain download. Includes the `subPath` trap and a phase timeline script. No GPU. |

## Prerequisites

- A Kubernetes cluster (kind, minikube, or managed)
- `kubectl` configured against your cluster
- Demo-specific tools listed in each directory's README

## Quick Start

```bash
# Pick a demo
cd knative/

# Follow the README inside
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to submit fixes or new demos.

## License

Apache 2.0 - see [LICENSE](LICENSE).

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

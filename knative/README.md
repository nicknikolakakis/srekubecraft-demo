# Knative Demo

Hands-on demo for the blog post [Knative - The Platform Engineer's Guide to Serverless on Kubernetes](https://srekubecraft.io/posts/knative/).

This demo deploys a full Knative stack on a local Kind cluster with Cilium CNI and walks through four scenarios: serving basics with scale-to-zero, autoscaling under load, canary traffic splitting, and event-driven architecture with CloudEvents.

## Architecture

```
Kind Cluster (3 nodes, Cilium CNI)
├── knative-serving (Kourier networking, KPA autoscaler)
├── knative-eventing (MT-Channel Broker, PingSource, ApiServerSource)
└── default namespace
    ├── hello-knative      (Demo 1: scale-to-zero)
    ├── autoscale-demo     (Demo 2: KPA autoscaling)
    ├── canary-demo        (Demo 3: traffic splitting)
    └── event-display      (Demo 4: CloudEvents sink)
```

## Prerequisites

| Tool | Version | Purpose |
| --- | --- | --- |
| [Docker](https://docs.docker.com/get-docker/) | 20+ | Container runtime |
| [Kind](https://kind.sigs.k8s.io/) | 0.20+ | Local Kubernetes cluster |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | 1.28+ | Kubernetes CLI |
| [Helm](https://helm.sh/docs/intro/install/) | 3.12+ | Package manager (Cilium, Metrics Server) |
| [Cilium CLI](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/#install-the-cilium-cli) | 0.15+ | CNI installation and status |
| [Flux CLI](https://fluxcd.io/flux/installation/) | 2.0+ | GitOps (optional setup path) |
| [kn CLI](https://knative.dev/docs/client/install-kn/) | 1.10+ | Knative service management |
| [Task](https://taskfile.dev/installation/) | 3.0+ | Task runner |

## Quick Start

```bash
git clone https://github.com/nicknikolakakis/srekubecraft-demo.git
cd srekubecraft-demo/knative

# Full setup: Kind cluster + Cilium + Knative + build app
task setup

# Or with Flux for GitOps-based Knative install
task setup:flux
```

## Demos

### Demo 1: Serving Basics and Scale-to-Zero

Deploys a Knative Service, tests it from inside the cluster, and demonstrates scale-to-zero behavior.

```bash
task demo:serving
```

**What it does:**
- Creates `hello-knative` service with `min-scale=0`, `max-scale=5`, `target=10`
- Sends a request via in-cluster curl
- Shows running pods

**Try it yourself:**
```bash
# Watch pods scale to zero after 60s of no traffic
kubectl get pods -l serving.knative.dev/service=hello-knative -w

# Trigger a cold start
kubectl run curl-test --rm -it --restart=Never --image=curlimages/curl \
  -- curl -s http://hello-knative.default.svc.cluster.local
```

### Demo 2: Autoscaling Under Load

Deploys a service with aggressive autoscaling settings and generates load with the `/burn` endpoint.

```bash
task demo:autoscale
```

**What it does:**
- Creates `autoscale-demo` with KPA, `target=5` concurrent requests, `max-scale=10`
- Tests the service from inside the cluster

**Try it yourself:**
```bash
# Terminal 1: Watch pods scale up
kubectl get pods -l serving.knative.dev/service=autoscale-demo -w

# Terminal 2: Generate load (50 concurrent, 30 seconds)
kubectl run load-test --rm -it --restart=Never --image=williamyeh/hey \
  -- -z 30s -c 50 http://autoscale-demo.default.svc.cluster.local/burn?duration=2
```

### Demo 3: Canary Deployment with Traffic Splitting

Deploys v1, updates to v2, and splits traffic 90/10 between revisions.

```bash
task demo:canary
```

**What it does:**
- Creates `canary-demo` v1 with `min-scale=1`
- Updates to v2 with 90% on v1, 10% on v2 (tagged `canary`)
- Sends 10 requests to show traffic distribution across revisions

**Try it yourself:**
```bash
# Promote v2 to 100% traffic
kn service update canary-demo --traffic @latest=100

# Verify
kn revision list
```

### Demo 4: Event-Driven Architecture

Sets up a complete eventing pipeline with Broker, PingSource, ApiServerSource, and Triggers.

```bash
task demo:eventing
```

**What it does:**
- Creates `event-display` sink service
- Deploys an MT-Channel Broker
- Creates a PingSource that sends a heartbeat every minute
- Creates an ApiServerSource that watches pod changes
- Routes both event types to the sink via Triggers

**Try it yourself:**
```bash
# Watch CloudEvents arriving at the sink
kubectl logs -l serving.knative.dev/service=event-display -c user-container -f
```

## Project Structure

```
knative/
├── app/
│   ├── main.go              # Go HTTP server (GET /, /burn, /healthz, POST / for CloudEvents)
│   ├── go.mod               # Go 1.26, stdlib only
│   ├── Dockerfile           # Multi-stage: golang:1.26-alpine -> distroless nonroot (14MB)
│   └── .dockerignore
├── kubernetes/
│   ├── cluster/
│   │   └── kind-config.yaml # 3-node Kind: Cilium-ready (no default CNI, no kube-proxy)
│   ├── cilium/
│   │   └── values.yaml      # Cilium Helm values (kubeProxyReplacement, Hubble)
│   ├── flux/
│   │   ├── sources.yaml     # HelmRepository (Cilium) + OCIRepository (Knative)
│   │   ├── cilium-release.yaml
│   │   ├── knative-serving.yaml   # CRDs -> Core -> Kourier (dependsOn chain)
│   │   ├── knative-eventing.yaml  # CRDs -> Core -> Channels -> Broker
│   │   └── namespace.yaml
│   ├── serving/
│   │   ├── 01-hello-service.yaml
│   │   ├── 02-autoscale-service.yaml
│   │   └── 03-traffic-split.yaml
│   └── eventing/
│       ├── 01-broker.yaml            # MT-Channel Broker
│       ├── 02-event-display.yaml     # Sink service
│       ├── 03-ping-source.yaml       # Cron heartbeat -> Broker
│       ├── 04-trigger.yaml           # Ping events -> event-display
│       ├── 05-api-server-source.yaml # Pod watcher -> Broker (RBAC included)
│       └── 06-pod-trigger.yaml       # Pod events -> event-display
├── Taskfile.yml
└── README.md
```

## Available Tasks

```
task --list
```

| Task | Description |
| --- | --- |
| `setup` | Full setup: Kind + Cilium + Knative + build app |
| `setup:flux` | Full setup using Flux for GitOps-based Knative install |
| `demo:serving` | Demo 1: Deploy a Knative Service and test scale-to-zero |
| `demo:autoscale` | Demo 2: Autoscaling with load generation |
| `demo:canary` | Demo 3: Canary deployment with traffic splitting |
| `demo:eventing` | Demo 4: Event-driven architecture with Broker, PingSource, and ApiServerSource |
| `knative:status` | Check Knative component status |
| `clean:services` | Delete all demo services |
| `clean` | Delete everything including the cluster |

## Stack Versions

| Component | Version |
| --- | --- |
| Knative Serving + Eventing | 1.17.0 |
| Cilium | 1.19.0 |
| Kourier (networking) | 1.17.0 |
| Go | 1.26 |

## Two Setup Paths

**Direct install** (`task setup`) installs Knative via `kubectl apply` from release manifests. Best for quick exploration.

**Flux install** (`task setup:flux`) installs Knative via Flux Kustomizations with proper dependency ordering. Best for understanding how you would run Knative in production with GitOps.

Both paths produce the same result: a Kind cluster with Cilium, Knative Serving (Kourier), and Knative Eventing ready for demos.

## Cleanup

```bash
# Remove demo services only (keep cluster)
task clean:services

# Delete everything
task clean
```

## Blog Post

This demo accompanies the blog post: [Knative - The Platform Engineer's Guide to Serverless on Kubernetes](https://srekubecraft.io/posts/knative/)

## License

See the repository root [LICENSE](../LICENSE) file.

# syntax=docker/dockerfile:1.7

# ---------------------------------------------------------------------------
# Stage 1: build the ADK agent binary.
# Run on the BUILDPLATFORM and cross-compile to TARGETPLATFORM so multi-arch
# builds (linux/amd64 + linux/arm64) avoid emulation overhead.
# ---------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# CGO off + osusergo/netgo build tags = no glibc, no NSS, no /etc/nsswitch.conf
# dependency at runtime. The result is a fully static binary safe for scratch.
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOFLAGS=-trimpath

# Cache module downloads independently of source changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Source last so edits do not invalidate the module cache.
# skills/ must be in the build context, since //go:embed bakes it into the binary.
COPY agent.go kubectl_tool.go ./
COPY skills/ ./skills/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
        -tags "osusergo,netgo" \
        -buildvcs=false \
        -ldflags="-s -w" \
        -o /out/agent ./

# ---------------------------------------------------------------------------
# Stage 2: assemble a minimal rootfs (CA bundle, /etc/passwd, /tmp, kubectl).
# Everything lives under /rootfs/ so the final stage can do one COPY.
# ---------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM alpine:3.23 AS rootfs

ARG KUBECTL_VERSION=v1.36.0
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache ca-certificates curl

RUN curl -fsSL --retry 3 \
        -o /usr/local/bin/kubectl \
        "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${TARGETOS}/${TARGETARCH}/kubectl" \
    && chmod 0755 /usr/local/bin/kubectl

# Lay out the exact directory tree we want copied into scratch.
RUN set -eux \
    && mkdir -p /rootfs/etc/ssl/certs /rootfs/usr/local/bin /rootfs/tmp /rootfs/home/nonroot /rootfs/app \
    && cp /etc/ssl/certs/ca-certificates.crt /rootfs/etc/ssl/certs/ca-certificates.crt \
    && cp /usr/local/bin/kubectl              /rootfs/usr/local/bin/kubectl \
    && echo 'nonroot:x:65532:65532:nonroot:/home/nonroot:/sbin/nologin' > /rootfs/etc/passwd \
    && echo 'nonroot:x:65532:'                                          > /rootfs/etc/group \
    && chown -R 65532:65532 /rootfs/tmp /rootfs/home/nonroot \
    && chmod 1777 /rootfs/tmp

# ---------------------------------------------------------------------------
# Stage 3: final scratch image. Empty base + only what the agent needs.
# ---------------------------------------------------------------------------
FROM scratch

COPY --from=rootfs /rootfs/ /
COPY --from=build  /out/agent /app/agent

WORKDIR /app
USER 65532:65532

ENV HOME=/home/nonroot \
    SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

ENTRYPOINT ["/app/agent"]
CMD ["web", "api", "webui"]

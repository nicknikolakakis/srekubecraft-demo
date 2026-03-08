# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Copy dependency manifests first for layer caching
COPY go.mod ./

# Copy source code
COPY main.go ./

# Build static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /server .

# Runtime stage - distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder --chown=65532:65532 /server /server

ENV PORT=8080

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/server"]

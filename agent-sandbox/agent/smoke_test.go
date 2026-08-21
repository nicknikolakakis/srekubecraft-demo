package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/agent-sandbox/clients/go/sandbox"
)

// integration smoke test: needs the demo cluster; run with SANDBOX_SMOKE=1
func TestSandboxSmoke(t *testing.T) {
	if os.Getenv("SANDBOX_SMOKE") != "1" {
		t.Skip("set SANDBOX_SMOKE=1 to run against the demo cluster")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	helper, err := sandbox.NewK8sHelper(nil, logr.Discard())
	if err != nil {
		t.Fatal(err)
	}
	client, err := sandbox.NewClient(ctx, sandbox.Options{Namespace: "default", K8sHelper: helper})
	if err != nil {
		t.Fatal(err)
	}
	defer client.DeleteAll(ctx)

	start := time.Now()
	sb, err := client.CreateSandbox(ctx, "python-sandbox-pool", "default")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sandbox ready in %s: claim=%s pod=%s", time.Since(start).Round(time.Millisecond), sb.ClaimName(), sb.PodName())

	lc := &lifecycle{helper: helper, namespace: "default", claim: sb.ClaimName(), ttl: 5 * time.Minute}
	if err := lc.extend(ctx); err != nil {
		t.Fatalf("extend TTL: %v", err)
	}

	res, err := sb.Run(ctx, "echo hello from $(hostname)")
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	t.Logf("run: %q", res.Stdout)

	if err := sb.Write(ctx, "smoke.txt", []byte("smoke-ok")); err != nil {
		t.Fatal(err)
	}
	data, err := sb.Read(ctx, "smoke.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "smoke-ok" {
		t.Fatalf("read back %q", data)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
	"github.com/go-logr/logr"
	"sigs.k8s.io/agent-sandbox/clients/go/sandbox"
)

const systemPrompt = `You are a coding agent with access to an isolated Linux sandbox running in Kubernetes.
Every tool call executes inside the sandbox pod, never on the user's machine.
Sandbox facts: Python 3 is available, the working directory is /app, and you
run as a non-root user (uid 1000). run_command does NOT go through a shell:
pipes, &&, redirection and $VARS are not interpreted. Wrap anything needing
shell features in: sh -c "your command here". write_file and read_file only
accept plain filenames (no directory separators) relative to the working
directory; use run_command for anything involving paths. Be concise: do the
task, verify the result by running it, then summarize what you did.`

func main() {
	task := flag.String("task", "", "task for the agent to perform")
	pool := flag.String("pool", "python-sandbox-pool", "SandboxWarmPool to claim from")
	namespace := flag.String("namespace", "default", "namespace for the SandboxClaim")
	ttl := flag.Duration("ttl", 5*time.Minute, "sandbox inactivity TTL, extended on every tool call")
	keep := flag.Bool("keep", false, "leave the sandbox running after the task (TTL still applies)")
	flag.Parse()
	if *task == "" {
		fmt.Fprintln(os.Stderr, `usage: sandbox-agent -task "..." [-pool ...] [-ttl 5m] [-keep]`)
		os.Exit(2)
	}

	ctx := context.Background()

	// own K8sHelper so we can also patch the claim's lifecycle directly
	helper, err := sandbox.NewK8sHelper(nil, logr.Discard())
	if err != nil {
		log.Fatal(err)
	}
	client, err := sandbox.NewClient(ctx, sandbox.Options{
		Namespace: *namespace,
		K8sHelper: helper,
	})
	if err != nil {
		log.Fatal(err)
	}
	stop := client.EnableAutoCleanup()
	defer stop()

	start := time.Now()
	sb, err := client.CreateSandbox(ctx, *pool, *namespace)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("sandbox ready in %s: claim=%s sandbox=%s pod=%s\n",
		time.Since(start).Round(time.Millisecond), sb.ClaimName(), sb.SandboxName(), sb.PodName())
	if !*keep {
		defer client.DeleteAll(ctx)
	}

	lc := &lifecycle{helper: helper, namespace: *namespace, claim: sb.ClaimName(), ttl: *ttl}
	if err := lc.extend(ctx); err != nil {
		log.Fatalf("arming sandbox TTL: %v", err)
	}

	tools, err := sandboxTools(sb, lc)
	if err != nil {
		log.Fatal(err)
	}

	llm := anthropic.NewClient()
	runner := llm.Beta.Messages.NewToolRunner(tools, anthropic.BetaToolRunnerParams{
		BetaMessageNewParams: anthropic.BetaMessageNewParams{
			Model:     "claude-opus-5",
			MaxTokens: 16000,
			System:    []anthropic.BetaTextBlockParam{{Text: systemPrompt}},
			Messages: []anthropic.BetaMessageParam{
				anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(*task)),
			},
			// re-serve policy declines with a fallback model in the same call
			Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
			Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		},
		MaxIterations: 20,
	})

	msg, err := runner.RunToCompletion(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if msg.StopReason == anthropic.BetaStopReasonRefusal {
		log.Fatalf("model refused: %s", msg.StopDetails.Explanation)
	}
	fmt.Println()
	for _, block := range msg.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			fmt.Println(b.Text)
		}
	}
	if *keep {
		fmt.Printf("\nsandbox kept alive: claim=%s (auto-deletes %s after last tool call)\n",
			sb.ClaimName(), *ttl)
	}
}

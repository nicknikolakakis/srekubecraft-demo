// Package main is a read-only Kubernetes operations assistant built with
// Google's Agent Development Kit (ADK) for Go.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/skilltoolset"
	"google.golang.org/adk/tool/skilltoolset/skill"
	"google.golang.org/genai"
)

// Bake the skills/ tree into the binary so the same code works locally and in a container.
//
//go:embed skills
var skillsFS embed.FS

// Read-only contract enforced at the prompt layer (defence in depth on top of the tool).
const systemInstruction = `You are a read-only Kubernetes operations assistant.

Hard rules:
- Use the "kubectl" tool ONLY with these verbs: get, describe, logs, top, events, explain, api-resources, api-versions, version, cluster-info, "config view".
- Never propose or attempt destructive actions (delete, edit, apply, exec, patch, scale, rollout undo, drain, cordon, label, annotate).
- When asked to fix something, suggest the change as a kubectl command or YAML diff for the user to run. Do not run it.
- Prefer namespaced queries. Ask for the namespace if it is unclear.
- For incident triage, follow this order: get pods, describe pod, logs (current and --previous), events, top.
- Keep answers concise. Use short bullet lists. Put commands in fenced code blocks.

Skills are available for common workflows (pod restart diagnosis, manifest review). Use them when the request matches.`

func main() {
	// Structured logger to stderr at INFO and above.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ctx := context.Background()

	// AI Studio path: read the Gemini API key from the environment.
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		slog.Error("GOOGLE_API_KEY is not set; export it or source .env first")
		os.Exit(1)
	}

	// Build the Gemini model client used by the LLM agent below.
	model, err := gemini.NewModel(ctx, "gemini-flash-latest", &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		slog.Error("failed to create model", "err", err)
		os.Exit(1)
	}

	// Wrap our restricted kubectl shell-out as an ADK function tool.
	kubectlTool, err := newKubectlTool()
	if err != nil {
		slog.Error("failed to create kubectl tool", "err", err)
		os.Exit(1)
	}

	// Strip the "skills" prefix so the FS root maps directly to skill names.
	skillsRoot, err := fs.Sub(skillsFS, "skills")
	if err != nil {
		slog.Error("failed to sub-mount skills FS", "err", err)
		os.Exit(1)
	}

	// Toolset that lazily loads SKILL.md files when the model selects one.
	skillSet, err := skilltoolset.New(ctx, skilltoolset.Config{
		Source: skill.NewFileSystemSource(skillsRoot),
	})
	if err != nil {
		slog.Error("failed to load skills", "err", err)
		os.Exit(1)
	}

	// Single LLM agent: model + tool + skills + system prompt.
	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "k8s_assistant",
		Model:       model,
		Description: "Read-only Kubernetes operations assistant.",
		Instruction: systemInstruction,
		Tools:       []tool.Tool{kubectlTool},
		Toolsets:    []tool.Toolset{skillSet},
	})
	if err != nil {
		slog.Error("failed to create agent", "err", err)
		os.Exit(1)
	}

	// Launcher dispatches CLI / API / web UI based on os.Args.
	cfg := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := full.NewLauncher()
	if err := l.Execute(ctx, cfg, os.Args[1:]); err != nil {
		slog.Error("run failed", "err", err, "syntax", l.CommandLineSyntax())
		os.Exit(1)
	}
}

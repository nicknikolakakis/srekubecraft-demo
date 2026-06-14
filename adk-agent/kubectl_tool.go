package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// Closed set of read-only kubectl subcommands we will execute.
var allowedVerbs = map[string]bool{
	"get":           true,
	"describe":      true,
	"logs":          true,
	"top":           true,
	"events":        true,
	"explain":       true,
	"api-resources": true,
	"api-versions":  true,
	"version":       true,
	"cluster-info":  true,
	// "config" is allowed; its sub-verb is whitelisted separately below.
	"config": true,
}

// `kubectl config <sub>` is restricted to inspection-only sub-verbs.
var allowedConfigSubverbs = map[string]bool{
	"view":            true,
	"current-context": true,
	"get-contexts":    true,
}

// Flags that could redirect kubectl to a different cluster or identity.
var blockedFlagPrefixes = []string{
	"--token",
	"--server",
	"--kubeconfig",
	"--as",
	"--as-group",
	"--client-key",
	"--client-certificate",
	"--username",
	"--password",
}

// JSON schema the model fills in when it calls the tool.
type kubectlInput struct {
	Verb      string   `json:"verb"                jsonschema:"kubectl verb. One of: get, describe, logs, top, events, explain, api-resources, api-versions, version, cluster-info, config."`
	Args      []string `json:"args,omitempty"      jsonschema:"Arguments after the verb. Example for get: [\"pods\", \"-o\", \"wide\"]. Do not include the verb here."`
	Namespace string   `json:"namespace,omitempty" jsonschema:"Optional namespace. If set, '-n <namespace>' is appended unless 'args' already contains '-n' or '--namespace'."`
}

// What we return to the model after a kubectl call.
type kubectlOutput struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

// Register runKubectl with ADK so the agent can invoke it.
func newKubectlTool() (tool.Tool, error) {
	return functiontool.New(functiontool.Config{
		Name: "kubectl",
		Description: "Run a read-only kubectl command. " +
			"Allowed verbs: get, describe, logs, top, events, explain, api-resources, api-versions, version, cluster-info, 'config view'. " +
			"Never use this tool for destructive operations.",
	}, runKubectl)
}

// Validate input, shell out to kubectl, capture and return output.
func runKubectl(_ tool.Context, in kubectlInput) (kubectlOutput, error) {
	// Reject anything outside the allowed verb set.
	verb := strings.TrimSpace(in.Verb)
	if !allowedVerbs[verb] {
		return kubectlOutput{}, fmt.Errorf(
			"verb %q is not allowed; permitted: get, describe, logs, top, events, explain, api-resources, api-versions, version, cluster-info, config",
			verb,
		)
	}

	// `config` only allows inspection sub-verbs.
	if verb == "config" {
		if len(in.Args) == 0 || !allowedConfigSubverbs[in.Args[0]] {
			return kubectlOutput{}, errors.New("'config' requires a sub-verb of: view, current-context, get-contexts")
		}
	}

	// Drop any argument that looks like a known-dangerous flag.
	for _, a := range in.Args {
		for _, bad := range blockedFlagPrefixes {
			if a == bad || strings.HasPrefix(a, bad+"=") {
				return kubectlOutput{}, fmt.Errorf("flag %q is not allowed", a)
			}
		}
	}

	// Final argv: verb first, then the model's args.
	cmdArgs := append([]string{verb}, in.Args...)

	// Default namespace flag when the model passed Namespace separately.
	if in.Namespace != "" && !hasNamespaceFlag(in.Args) {
		cmdArgs = append(cmdArgs, "-n", in.Namespace)
	}

	// Hard 30s timeout so a hung kubectl can't stall the agent.
	cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, "kubectl", cmdArgs...)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Pretty-printed command we echo back to the model for traceability.
	rendered := "kubectl " + strings.Join(cmdArgs, " ")

	// exec.ExitError = process ran but exited non-zero (still a useful tool result).
	err := cmd.Run()
	exit := 0
	var ee *exec.ExitError
	switch {
	case err == nil:
		// success
	case errors.As(err, &ee):
		exit = ee.ExitCode()
	default:
		// kubectl missing, deadline exceeded, etc. Hand back as a tool result so the LLM can react.
		return kubectlOutput{
			Command:  rendered,
			ExitCode: -1,
			Stderr:   err.Error(),
		}, nil
	}

	// Cap output sizes to keep the model's context tight.
	return kubectlOutput{
		Command:  rendered,
		ExitCode: exit,
		Stdout:   truncate(stdout.String(), 16*1024),
		Stderr:   truncate(stderr.String(), 4*1024),
	}, nil
}

// True if args already include -n / --namespace.
func hasNamespaceFlag(args []string) bool {
	for _, a := range args {
		if a == "-n" || a == "--namespace" || strings.HasPrefix(a, "--namespace=") {
			return true
		}
	}
	return false
}

// Cap s at n chars and append a marker if cut.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n... [truncated]"
}

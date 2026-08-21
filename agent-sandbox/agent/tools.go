package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/toolrunner"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/agent-sandbox/clients/go/sandbox"
	extensionsv1beta1 "sigs.k8s.io/agent-sandbox/extensions/api/v1beta1"
)

// lifecycle pushes the claim's shutdownTime forward on every tool call, so an
// idle or crashed session is garbage-collected by the controller, not by us.
type lifecycle struct {
	helper    *sandbox.K8sHelper
	namespace string
	claim     string
	ttl       time.Duration
}

func (l *lifecycle) extend(ctx context.Context) error {
	// the controller also updates the claim, so retry on 409 conflicts
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		claim, err := l.helper.ExtensionsClient.SandboxClaims(l.namespace).Get(ctx, l.claim, metav1.GetOptions{})
		if err != nil {
			return err
		}
		shutdownAt := metav1.NewTime(time.Now().Add(l.ttl))
		claim.Spec.Lifecycle = &extensionsv1beta1.Lifecycle{
			ShutdownPolicy: extensionsv1beta1.ShutdownPolicyDelete,
			ShutdownTime:   &shutdownAt,
		}
		_, err = l.helper.ExtensionsClient.SandboxClaims(l.namespace).Update(ctx, claim, metav1.UpdateOptions{})
		return err
	})
}

type runCommandInput struct {
	Command string `json:"command" jsonschema:"required,description=Shell command to execute inside the sandbox"`
}

type writeFileInput struct {
	Filename string `json:"filename" jsonschema:"required,description=Plain filename without directory separators"`
	Content  string `json:"content" jsonschema:"required,description=Full content to write"`
}

type readFileInput struct {
	Filename string `json:"filename" jsonschema:"required,description=Plain filename without directory separators"`
}

type listFilesInput struct {
	Path string `json:"path" jsonschema:"description=Directory to list. Defaults to the working directory"`
}

func textResult(s string) (anthropic.BetaToolResultBlockParamContentUnion, error) {
	return anthropic.BetaToolResultBlockParamContentUnion{
		OfText: &anthropic.BetaTextBlockParam{Text: s},
	}, nil
}

func sandboxTools(sb *sandbox.Sandbox, lc *lifecycle) ([]anthropic.BetaTool, error) {
	touch := func(ctx context.Context) {
		if err := lc.extend(ctx); err != nil {
			log.Printf("warning: extending sandbox TTL: %v", err)
		}
	}

	runCommand, err := toolrunner.NewBetaToolFromJSONSchema(
		"run_command",
		"Execute a shell command inside the sandbox. Returns stdout, stderr and the exit code.",
		func(ctx context.Context, in runCommandInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			fmt.Printf("[tool] run_command: %s\n", in.Command)
			touch(ctx)
			res, err := sb.Run(ctx, in.Command)
			if err != nil {
				return textResult(fmt.Sprintf("execution error: %v", err))
			}
			return textResult(fmt.Sprintf("exit_code: %d\nstdout:\n%s\nstderr:\n%s",
				res.ExitCode, res.Stdout, res.Stderr))
		})
	if err != nil {
		return nil, err
	}

	writeFile, err := toolrunner.NewBetaToolFromJSONSchema(
		"write_file",
		"Write a file in the sandbox working directory. Accepts plain filenames only.",
		func(ctx context.Context, in writeFileInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			fmt.Printf("[tool] write_file: %s (%d bytes)\n", in.Filename, len(in.Content))
			touch(ctx)
			if err := sb.Write(ctx, in.Filename, []byte(in.Content)); err != nil {
				return textResult(fmt.Sprintf("write error: %v", err))
			}
			return textResult("written " + in.Filename)
		})
	if err != nil {
		return nil, err
	}

	readFile, err := toolrunner.NewBetaToolFromJSONSchema(
		"read_file",
		"Read a file from the sandbox working directory. Accepts plain filenames only.",
		func(ctx context.Context, in readFileInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			fmt.Printf("[tool] read_file: %s\n", in.Filename)
			touch(ctx)
			data, err := sb.Read(ctx, in.Filename)
			if err != nil {
				return textResult(fmt.Sprintf("read error: %v", err))
			}
			return textResult(string(data))
		})
	if err != nil {
		return nil, err
	}

	listFiles, err := toolrunner.NewBetaToolFromJSONSchema(
		"list_files",
		"List files and directories in the sandbox.",
		func(ctx context.Context, in listFilesInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			path := in.Path
			if path == "" {
				path = "."
			}
			fmt.Printf("[tool] list_files: %s\n", path)
			touch(ctx)
			entries, err := sb.List(ctx, path)
			if err != nil {
				return textResult(fmt.Sprintf("list error: %v", err))
			}
			out := ""
			for _, e := range entries {
				out += fmt.Sprintf("%s\t%d\t%s\n", e.Type, e.Size, e.Name)
			}
			if out == "" {
				out = "(empty)"
			}
			return textResult(out)
		})
	if err != nil {
		return nil, err
	}

	return []anthropic.BetaTool{runCommand, writeFile, readFile, listFiles}, nil
}

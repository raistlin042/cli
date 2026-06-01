// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
)

// commandRunner abstracts external process execution so apps +init's
// orchestration can be unit-tested without a real git binary or network.
// dir == "" runs in the current working directory; a non-empty dir runs the
// command with that working directory (git -C semantics).
type commandRunner interface {
	Run(ctx context.Context, dir, name string, args ...string) (stdout, stderr string, err error)
}

// execCommandRunner is the production commandRunner backed by os/exec.
type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, dir, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// credentialURLRe matches the userinfo segment of an http(s) URL (the
// "user:token@" part) so it can be redacted before any output or logging.
var credentialURLRe = regexp.MustCompile(`((?:https?)://)[^/@\s]+@`)

// redactURLCredentials replaces the userinfo segment of any http(s) URL in s
// with "***". Safe to call on both a bare repo_url and free-form text such as
// git stderr (which echoes the full remote URL on failure).
func redactURLCredentials(s string) string {
	return credentialURLRe.ReplaceAllString(s, "${1}***@")
}

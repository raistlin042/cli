// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

// defaultInitBranch is the fixed remote branch +init checks out after clone.
const defaultInitBranch = "sprint/default"

// initCommitMessage is the fixed commit subject used when the post-init working
// tree has changes to push. Fixed constant — never interpolates user input.
const initCommitMessage = "chore: scaffold app via lark-cli apps +init"

// TODO(apps-init): run the npx scaffold command here once it is defined.
// const initNpxCommand = "npx <scaffold-cmd>"

// initRunner is the commandRunner used by +init. Package-level so unit tests
// can swap in a fakeCommandRunner. Production uses execCommandRunner.
var initRunner commandRunner = execCommandRunner{}

// AppsInit initializes a Miaoda app's local development repository.
var AppsInit = common.Shortcut{
	Service:     appsService,
	Command:     "+init",
	Description: "Initialize a Miaoda app's local development repository",
	Risk:        "write",
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		// NOTE: --app-id is intentionally NOT Required:true. The framework maps
		// Required:true to cobra's MarkFlagRequired, whose error is plain-text
		// exit-1 (root.go handleRootError case 4), bypassing the structured
		// envelope. The spec and the E2E assert exit-2 + a structured
		// {"ok":false,"error":{...}} envelope for missing --app-id, so the empty
		// check lives in Validate (output.ErrValidation -> ExitValidation=2).
		{Name: "app-id", Desc: "Miaoda app ID"},
		{Name: "dir", Desc: "clone target directory (default ./<app-id>)"},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		dir := strings.TrimSpace(rctx.Str("dir"))
		if dir == "" {
			dir = defaultCloneDir(appID)
		}
		return common.NewDryRunAPI().
			Desc("Initialize Miaoda app repository (orchestrates credential-init, git clone, checkout, optional commit/push)").
			Set("credential_init", fmt.Sprintf("apps +git-credential-init --app-id %s --format json", appID)).
			Set("clone", fmt.Sprintf("git clone -- <repo_url-from-credential-init> %s", dir)).
			Set("checkout", "git checkout "+defaultInitBranch).
			Set("commit_push", "conditional: git add -A + commit + push origin "+defaultInitBranch+" only when the working tree has changes (npx step skipped this release)").
			Set("clone_path", dir).
			Set("npx_skipped", true)
	},
	Execute: appsInitExecute,
}

// defaultCloneDir returns the default clone target (./<app-id>) for an app ID.
func defaultCloneDir(appID string) string {
	return filepath.Join(".", appID)
}

// resolveCloneDir computes the absolute clone target from --dir (or the
// ./<app-id> default), validates it against path-traversal, and refuses to
// clone into an existing non-empty directory.
func resolveCloneDir(rctx *common.RuntimeContext, appID string) (string, error) {
	raw := strings.TrimSpace(rctx.Str("dir"))
	if raw == "" {
		raw = defaultCloneDir(appID)
	}
	safe, err := validate.SafeInputPath(raw)
	if err != nil {
		return "", output.ErrValidation("--dir is not a safe path: %v", err)
	}
	abs, err := filepath.Abs(safe)
	if err != nil {
		return "", output.ErrValidation("--dir cannot be resolved: %v", err)
	}
	if err := ensureEmptyDir(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// ensureEmptyDir returns a validation error if dir exists and is non-empty.
// A non-existent path is fine — git clone creates it.
func ensureEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return output.ErrValidation("--dir cannot be read: %v", err)
	}
	if len(entries) > 0 {
		return output.ErrValidation("target directory %q already exists and is not empty", dir)
	}
	return nil
}

// appsInitExecute is a stub this task; Task 3 replaces the body with the real
// orchestration.
func appsInitExecute(ctx context.Context, rctx *common.RuntimeContext) error { return nil }

// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/larksuite/cli/internal/charcheck"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
)

// defaultInitBranch is the fixed remote branch +init checks out after clone.
const defaultInitBranch = "sprint/default"

// initCommitMessage is the fixed commit subject used when the post-init working
// tree has changes to push. Fixed constant — never interpolates user input.
const initCommitMessage = "chore: scaffold app via lark-cli apps +init"

const (
	miaodaCLIPkg    = "@lark-apaas/miaoda-cli@alpha"
	defaultTemplate = "nestjs-react-fullstack"
	metaRelPath     = ".spark/meta.json"
	steeringRelPath = ".agent/skills/steering"
)

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
	// +init makes no direct lark API calls (it shells out to the
	// +git-credential-init subprocess, which enforces its own scopes), so it
	// declares no scopes of its own. Explicit []string{} (not nil) per the
	// convention enforced by TestAllShortcutsScopesNotNil.
	Scopes:    []string{},
	AuthTypes: []string{"user"},
	HasFormat: true,
	Flags: []common.Flag{
		// NOTE: --app-id is intentionally NOT Required:true. The framework maps
		// Required:true to cobra's MarkFlagRequired, whose error is plain-text
		// exit-1 (root.go handleRootError case 4), bypassing the structured
		// envelope. The spec and the E2E assert exit-2 + a structured
		// {"ok":false,"error":{...}} envelope for missing --app-id, so the empty
		// check lives in Validate (output.ErrValidation -> ExitValidation=2).
		{Name: "app-id", Desc: "Miaoda app ID"},
		{Name: "dir", Desc: "clone target directory; absolute or relative path (default ./<app-id>)"},
		{Name: "template", Default: defaultTemplate, Desc: "scaffold template used for an empty repo (app init)"},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		dry := common.NewDryRunAPI().
			Desc("Initialize Miaoda app repository (orchestrates credential-init, git clone, checkout, optional commit/push)").
			Set("credential_init", fmt.Sprintf("apps +git-credential-init --app-id %s --format json", appID)).
			Set("checkout", "git checkout "+defaultInitBranch).
			Set("commit_push", "conditional: git add -A + commit + push origin "+defaultInitBranch+" only when the working tree has changes (npx step skipped this release)").
			Set("npx_skipped", true)
		dir, err := resolveTargetPath(rctx, appID)
		if err == nil {
			err = ensureEmptyDir(dir)
		}
		if err != nil {
			// Advisory preview: surface the rejection as a field so --dry-run still exits 0.
			dry.Set("dir_error", err.Error())
			dir = defaultCloneDir(appID)
		}
		dry.Set("clone", fmt.Sprintf("git clone -- <repository_url-from-credential-init> %s", dir))
		dry.Set("clone_path", dir)
		return dry
	},
	Execute: appsInitExecute,
}

// defaultCloneDir returns the default clone target (./<app-id>) for an app ID.
func defaultCloneDir(appID string) string {
	return filepath.Join(".", appID)
}

// resolveTargetPath computes the absolute clone target from --dir (or the
// ./<app-id> default). Unlike the prior SafeInputPath approach it does NOT
// confine to cwd — the clone destination is user-chosen (the skill prompts for
// it). It rejects empty input and control characters; symlink/no-clobber
// guarding happens in ensureEmptyDir.
func resolveTargetPath(rctx *common.RuntimeContext, appID string) (string, error) {
	raw := strings.TrimSpace(rctx.Str("dir"))
	if raw == "" {
		raw = defaultCloneDir(appID)
	}
	if err := charcheck.RejectControlChars(raw, "--dir"); err != nil {
		return "", output.ErrValidation("%v", err)
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", output.ErrValidation("--dir cannot be resolved: %v", err)
	}
	return abs, nil
}

// ensureEmptyDir refuses to clone into an existing non-empty dir, a symlink, or
// a non-directory. A non-existent path is fine (git clone creates it). Uses
// Lstat so a symlinked target is rejected rather than followed.
func ensureEmptyDir(dir string) error {
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return output.ErrValidation("--dir cannot be read: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return output.ErrValidation("--dir must not be a symlink: %q", dir)
	}
	if !info.IsDir() {
		return output.ErrValidation("--dir exists and is not a directory: %q", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return output.ErrValidation("--dir cannot be read: %v", err)
	}
	if len(entries) > 0 {
		return output.ErrValidation("target directory %q already exists and is not empty", dir)
	}
	return nil
}

// isAlreadyInitialized reports whether dir is an already-initialized Miaoda app
// repo, detected by the presence of <dir>/.spark/meta.json (regardless of its
// app_id value). Used to short-circuit +init into a friendly no-op.
func isAlreadyInitialized(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, metaRelPath))
	return err == nil && !info.IsDir()
}

// ensureMetaAppID patches <dir>/.spark/meta.json to include app_id when the file
// exists but lacks (or has an empty) app_id. Other fields are preserved. When
// the file does not exist, this is a no-op (we never create it).
func ensureMetaAppID(dir, appID string) error {
	path := filepath.Join(dir, metaRelPath)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return output.Errorf(output.ExitAPI, "meta_write", "read %s failed: %v", metaRelPath, err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return output.Errorf(output.ExitAPI, "meta_write", "parse %s failed: %v", metaRelPath, err)
	}
	if cur, _ := m["app_id"].(string); strings.TrimSpace(cur) != "" {
		return nil
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	m["app_id"] = appID
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return output.Errorf(output.ExitAPI, "meta_write", "marshal %s failed: %v", metaRelPath, err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return output.Errorf(output.ExitAPI, "meta_write", "write %s failed: %v", metaRelPath, err)
	}
	return nil
}

// hasSteeringSkills reports whether <dir>/.agent/skills/steering exists as a dir.
func hasSteeringSkills(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, steeringRelPath))
	return err == nil && info.IsDir()
}

// isEmptyRepo reports whether the checked-out branch has zero tracked files,
// via `git ls-files`. Empty output = empty repo.
func isEmptyRepo(ctx context.Context, dir string) (bool, error) {
	stdout, stderr, err := initRunner.Run(ctx, dir, "git", "ls-files")
	if err != nil {
		return false, output.Errorf(output.ExitAPI, "git_ls_files", "git ls-files failed: %s", gitErr(stderr, err))
	}
	return strings.TrimSpace(stdout) == "", nil
}

// runScaffold runs the npx scaffolding step inside the cloned repo (cwd=dir).
// Empty repo -> `app init`; non-empty -> `app upgrade` + meta app_id patch +
// conditional `skills sync`. Returns "init" or "upgrade".
func runScaffold(ctx context.Context, dir, appID, template string) (string, error) {
	empty, err := isEmptyRepo(ctx, dir)
	if err != nil {
		return "", err
	}
	if empty {
		// TODO(apps-init): emptiness via `git ls-files` assumes the backend creates
		// a truly empty repo (no seed files like README/.gitignore). If seed files
		// can appear, switch to a marker-file check (e.g. package.json presence).
		// Confirm the backend's new-repo contents.
		if _, stderr, err := initRunner.Run(ctx, dir, "npx", miaodaCLIPkg, "app", "init", "--template", template, "--app-id", appID); err != nil {
			return "", output.Errorf(output.ExitAPI, "npx_app_init", "npx app init failed: %s", gitErr(stderr, err))
		}
		return "init", nil
	}
	if _, stderr, err := initRunner.Run(ctx, dir, "npx", miaodaCLIPkg, "app", "upgrade"); err != nil {
		return "", output.Errorf(output.ExitAPI, "npx_app_upgrade", "npx app upgrade failed: %s", gitErr(stderr, err))
	}
	if err := ensureMetaAppID(dir, appID); err != nil {
		return "", err
	}
	if !hasSteeringSkills(dir) {
		if _, stderr, err := initRunner.Run(ctx, dir, "npx", miaodaCLIPkg, "skills", "sync"); err != nil {
			return "", output.Errorf(output.ExitAPI, "npx_skills_sync", "npx skills sync failed: %s", gitErr(stderr, err))
		}
	}
	return "upgrade", nil
}

// parseRepoURLFromEnvelope extracts data.repository_url from a lark-cli JSON
// envelope ({"ok":true,"data":{"repository_url":"..."}}). The field name
// matches the contract emitted by `apps +git-credential-init`.
func parseRepoURLFromEnvelope(stdout string) (string, error) {
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			RepositoryURL string `json:"repository_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		return "", output.Errorf(output.ExitInternal, "credential_init", "could not parse +git-credential-init output as JSON: %v", err)
	}
	if !env.OK {
		return "", output.Errorf(output.ExitInternal, "credential_init", "+git-credential-init reported failure")
	}
	if strings.TrimSpace(env.Data.RepositoryURL) == "" {
		return "", output.Errorf(output.ExitInternal, "credential_init", "+git-credential-init returned no repository_url")
	}
	return env.Data.RepositoryURL, nil
}

// validateRepoURLScheme rejects any repository_url that is not http(s):// to
// block git's dangerous transports (ext::, file://, ssh://) and option injection.
func validateRepoURLScheme(repoURL string) error {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		return nil
	}
	return output.Errorf(output.ExitValidation, "validation",
		"repository_url from +git-credential-init must be http(s); refusing %q", redactURLCredentials(repoURL))
}

func appsInitExecute(ctx context.Context, rctx *common.RuntimeContext) error {
	appID := strings.TrimSpace(rctx.Str("app-id"))

	if _, err := exec.LookPath("git"); err != nil {
		return output.ErrWithHint(output.ExitInternal, "dependency",
			"git executable not found on PATH", "install git and ensure it is on your PATH")
	}

	dir, err := resolveTargetPath(rctx, appID)
	if err == nil {
		err = ensureEmptyDir(dir)
	}
	if err != nil {
		return err
	}

	repoURL, err := issueCredentials(ctx, rctx, appID)
	if err != nil {
		return err
	}
	if err := validateRepoURLScheme(repoURL); err != nil {
		return err
	}

	if _, stderr, err := initRunner.Run(ctx, "", "git", "clone", "--", repoURL, dir); err != nil {
		return output.Errorf(output.ExitAPI, "git_clone", "git clone failed: %s", gitErr(stderr, err))
	}

	if _, stderr, err := initRunner.Run(ctx, dir, "git", "checkout", defaultInitBranch); err != nil {
		return output.Errorf(output.ExitAPI, "git_checkout", "git checkout %s failed: %s", defaultInitBranch, gitErr(stderr, err))
	}

	// npx step skipped this release.
	// TODO(apps-init): run the npx scaffold command here once defined.

	committed, pushed, err := commitAndPushIfDirty(ctx, dir)
	if err != nil {
		return err
	}

	out := map[string]interface{}{
		"app_id":         appID,
		"repository_url": redactURLCredentials(repoURL),
		"branch":         defaultInitBranch,
		"clone_path":     dir,
		"committed":      committed,
		"pushed":         pushed,
		"npx_skipped":    true,
		"message":        "Repository initialized. You can start developing.",
	}
	rctx.OutFormat(out, nil, func(w io.Writer) {
		fmt.Fprintf(w, "✓ Repository initialized at %s\n", dir)
		fmt.Fprintf(w, "  branch: %s\n", defaultInitBranch)
		fmt.Fprintln(w, "仓库已初始化完成，可以开始开发了。")
	})
	return nil
}

// issueCredentials runs `<self> apps +git-credential-init --app-id <id> --format json`
// and returns the repo_url it reports. Forwards --as when set.
func issueCredentials(ctx context.Context, rctx *common.RuntimeContext, appID string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", output.Errorf(output.ExitInternal, "internal", "cannot locate lark-cli executable: %v", err)
	}
	args := []string{"apps", "+git-credential-init", "--app-id", appID, "--format", "json"}
	if as := strings.TrimSpace(rctx.Str("as")); as != "" {
		args = append(args, "--as", as)
	}
	stdout, stderr, err := initRunner.Run(ctx, "", self, args...)
	if err != nil {
		return "", output.ErrWithHint(output.ExitAPI, "credential_init",
			fmt.Sprintf("apps +git-credential-init failed: %s", gitErr(stderr, err)),
			"ensure apps +git-credential-init is available and you are logged in")
	}
	return parseRepoURLFromEnvelope(stdout)
}

// commitAndPushIfDirty commits and pushes only when the working tree has
// changes; a clean tree is a no-op (returns false,false).
func commitAndPushIfDirty(ctx context.Context, dir string) (committed, pushed bool, err error) {
	status, stderr, runErr := initRunner.Run(ctx, dir, "git", "status", "--porcelain")
	if runErr != nil {
		return false, false, output.Errorf(output.ExitAPI, "git_status", "git status failed: %s", gitErr(stderr, runErr))
	}
	if strings.TrimSpace(status) == "" {
		return false, false, nil
	}
	if _, se, e := initRunner.Run(ctx, dir, "git", "add", "-A"); e != nil {
		return false, false, output.Errorf(output.ExitAPI, "git_add", "git add failed: %s", gitErr(se, e))
	}
	if _, se, e := initRunner.Run(ctx, dir, "git", "commit", "-m", initCommitMessage); e != nil {
		return false, false, output.Errorf(output.ExitAPI, "git_commit", "git commit failed: %s", gitErr(se, e))
	}
	if _, se, e := initRunner.Run(ctx, dir, "git", "push", "origin", defaultInitBranch); e != nil {
		return true, false, output.Errorf(output.ExitAPI, "git_push", "git push failed: %s", gitErr(se, e))
	}
	return true, true, nil
}

// gitErr builds a redacted, single-line error detail from stderr (falling back
// to the exec error). Always redacts embedded credentials.
func gitErr(stderr string, err error) string {
	s := strings.TrimSpace(stderr)
	if s == "" && err != nil {
		s = err.Error()
	}
	return redactURLCredentials(s)
}

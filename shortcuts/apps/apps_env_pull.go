// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

// AppsEnvPull pulls startup env vars for an app into the local .env.local file.
var AppsEnvPull = common.Shortcut{
	Service:           appsService,
	Command:           "+env-pull",
	Description:       "Pull app startup env vars into the local project .env.local",
	Risk:              "write",
	Scopes:            []string{},
	ConditionalScopes: []string{"spark:app:read"},
	AuthTypes:         []string{"user"},
	HasFormat:         true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "app ID"},
		{Name: "project-path", Desc: "local project root path (defaults to current directory)"},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: "--app-id is required"}, Param: "app-id"}
		}
		_, envFile, err := resolveEnvPullTarget(strings.TrimSpace(rctx.Str("project-path")))
		if err != nil {
			return &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: fmt.Sprintf("--project-path: %v", err)}, Param: "project-path", Cause: err}
		}
		if err := checkEnvPullTarget(envFile); err != nil {
			return err
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		projectPath, envFile, _ := resolveEnvPullTarget(strings.TrimSpace(rctx.Str("project-path")))
		appID := strings.TrimSpace(rctx.Str("app-id"))
		return common.NewDryRunAPI().
			POST(fmt.Sprintf("%s/apps/%s/env_vars", apiBasePath, validate.EncodePathSegment(appID))).
			Desc("Pull app startup env vars into the local .env.local file").
			Set("project_path", projectPath).
			Set("env_file", envFile)
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		projectPath, envFile, err := resolveEnvPullTarget(strings.TrimSpace(rctx.Str("project-path")))
		if err != nil {
			return &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: fmt.Sprintf("--project-path: %v", err)}, Param: "project-path", Cause: err}
		}
		if err := checkEnvPullTarget(envFile); err != nil {
			return err
		}
		if err := rctx.EnsureScopes([]string{"spark:app:read"}); err != nil {
			return err
		}

		path := fmt.Sprintf("%s/apps/%s/env_vars", apiBasePath, validate.EncodePathSegment(appID))
		data, err := rctx.CallAPI("POST", path, nil, nil)
		if err != nil {
			return err
		}

		envVars, err := extractEnvPullVars(data)
		if err != nil {
			return err
		}
		original, err := readEnvPullFile(envFile)
		if err != nil {
			return err
		}
		merged, updated, created := mergeEnvPullFileContent(original, envVars)
		if err := ensureEnvPullParentDir(envFile); err != nil {
			return err
		}
		if err := validate.AtomicWrite(envFile, []byte(merged), 0o600); err != nil {
			return &errs.InternalError{Problem: errs.Problem{Category: errs.CategoryInternal, Message: fmt.Sprintf("cannot write %s: %v", envFile, err)}, Cause: err}
		}

		result := buildEnvPullSuccessData(appID, projectPath, envFile, hasEnvPullDatabase(envVars), len(updated), len(created))
		rctx.OutFormat(result, nil, func(w io.Writer) {
			writeEnvPullPretty(w, appID, envFile, hasEnvPullDatabase(envVars))
		})
		return nil
	},
}

func resolveEnvPullTarget(projectPath string) (string, string, error) {
	if strings.TrimSpace(projectPath) == "" {
		cwd, err := os.Getwd() //nolint:forbidigo // shortcuts cannot import internal/vfs; cwd lookup is local-only and bounded.
		if err != nil {
			return "", "", fmt.Errorf("cannot determine working directory: %w", err)
		}
		projectPath = cwd
	}
	if err := validate.RejectControlChars(projectPath, "--project-path"); err != nil {
		return "", "", err
	}
	projectPath = filepath.Clean(projectPath)
	return projectPath, filepath.Join(projectPath, ".env.local"), nil
}

func checkEnvPullTarget(envFile string) error {
	info, err := os.Lstat(envFile) //nolint:forbidigo // shortcuts cannot import internal/vfs; direct lstat is needed to reject symlinks before write.
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: fmt.Sprintf("cannot inspect %s: %v", envFile, err)}, Param: "project-path", Cause: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: fmt.Sprintf("target %s must be a regular file, not a symlink", envFile)}, Param: "project-path"}
	}
	if !info.Mode().IsRegular() {
		return &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: fmt.Sprintf("target %s must be a regular file", envFile)}, Param: "project-path"}
	}
	return nil
}

func extractEnvPullVars(data map[string]interface{}) (map[string]string, error) {
	raw := data["env_vars"]
	if raw == nil {
		if nested, ok := data["data"].(map[string]interface{}); ok {
			raw = nested["env_vars"]
		}
	}
	if raw == nil {
		return nil, &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: "response field env_vars must be an object of string values"}}
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: "response field env_vars must be an object of string values"}}
	}
	out := make(map[string]string, len(obj))
	for key, value := range obj {
		s, ok := value.(string)
		if !ok {
			return nil, &errs.ValidationError{Problem: errs.Problem{Category: errs.CategoryValidation, Message: "response field env_vars must be an object of string values"}}
		}
		out[key] = s
	}
	return out, nil
}

func readEnvPullFile(envFile string) (string, error) {
	data, err := os.ReadFile(envFile) //nolint:forbidigo // shortcuts cannot import internal/vfs; validated local file read for a single env file.
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", &errs.InternalError{Problem: errs.Problem{Category: errs.CategoryInternal, Message: fmt.Sprintf("cannot read %s: %v", envFile, err)}, Cause: err}
	}
	return string(data), nil
}

func ensureEnvPullParentDir(envFile string) error {
	dir := filepath.Dir(envFile)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:forbidigo // shortcuts cannot import internal/vfs; local mkdir for target env parent dir.
		return &errs.InternalError{Problem: errs.Problem{Category: errs.CategoryInternal, Message: fmt.Sprintf("cannot create %s: %v", dir, err)}, Cause: err}
	}
	return nil
}

func mergeEnvPullFileContent(original string, envVars map[string]string) (string, []string, []string) {
	if len(envVars) == 0 {
		if original == "" {
			return "", nil, nil
		}
		return ensureTrailingNewline(original), nil, nil
	}

	normalized := strings.ReplaceAll(original, "\r\n", "\n")
	lines := []string{}
	if normalized != "" {
		lines = strings.Split(normalized, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	}

	used := make(map[string]bool, len(envVars))
	updated := make([]string, 0, len(envVars))
	for i, line := range lines {
		key, ok := parseEnvPullAssignmentLine(line)
		if !ok {
			continue
		}
		value, exists := envVars[key]
		if !exists {
			continue
		}
		lines[i] = formatEnvPullAssignment(key, value)
		updated = append(updated, key)
		used[key] = true
	}

	created := make([]string, 0, len(envVars))
	pending := make([]string, 0, len(envVars))
	for key := range envVars {
		if used[key] {
			continue
		}
		pending = append(pending, key)
	}
	sort.Strings(pending)
	for _, key := range pending {
		lines = append(lines, formatEnvPullAssignment(key, envVars[key]))
		created = append(created, key)
	}

	sort.Strings(updated)
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return content, updated, created
}

func parseEnvPullAssignmentLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	idx := strings.Index(trimmed, "=")
	if idx <= 0 {
		return "", false
	}
	key := strings.TrimSpace(trimmed[:idx])
	value := strings.TrimSpace(trimmed[idx+1:])
	if key == "" || len(value) < 2 || !strings.HasPrefix(value, `"`) || !strings.HasSuffix(value, `"`) {
		return "", false
	}
	return key, true
}

func formatEnvPullAssignment(key, value string) string {
	return fmt.Sprintf("%s = %s", key, strconv.Quote(value))
}

func buildEnvPullSuccessData(appID, projectPath, envFile string, databaseDetected bool, updatedCount, createdCount int) map[string]interface{} {
	return map[string]interface{}{
		"app_id":            appID,
		"project_path":      projectPath,
		"env_file":          envFile,
		"database_detected": databaseDetected,
		"updated_count":     updatedCount,
		"created_count":     createdCount,
	}
}

func hasEnvPullDatabase(envVars map[string]string) bool {
	_, ok := envVars["SUDA_DATABASE_URL"]
	return ok
}

func writeEnvPullPretty(w io.Writer, appID, envFile string, databaseDetected bool) {
	fmt.Fprintf(w, "✓ App detected: %s\n", appID)
	if databaseDetected {
		fmt.Fprintln(w, "✓ Development database detected")
	}
	fmt.Fprintf(w, "✓ Local environment written to %s. Run `lark-cli apps +env-pull` again to refresh it.\n", envFile)
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

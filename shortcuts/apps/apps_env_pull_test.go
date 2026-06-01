// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/internal/output"
)

func assertValidationError(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if !errs.IsValidation(err) && output.ExitCodeOf(err) != output.ExitValidation {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
	if wantSubstr != "" && !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected validation message containing %q, got %v", wantSubstr, err)
	}
}

func TestResolveEnvPullTarget_DefaultProjectPathUsesCWD(t *testing.T) {
	cwd := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() err=%v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() err=%v", err)
	}
	wantProject, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() after Chdir err=%v", err)
	}

	gotProject, gotFile, err := resolveEnvPullTarget("")
	if err != nil {
		t.Fatalf("resolveEnvPullTarget() err=%v", err)
	}
	if gotProject != wantProject {
		t.Fatalf("project path = %q, want %q", gotProject, wantProject)
	}
	wantFile := filepath.Join(wantProject, ".env.local")
	if gotFile != wantFile {
		t.Fatalf("env file = %q, want %q", gotFile, wantFile)
	}
}

func TestResolveEnvPullTarget_CustomProjectPath(t *testing.T) {
	root := t.TempDir()
	gotProject, gotFile, err := resolveEnvPullTarget(root)
	if err != nil {
		t.Fatalf("resolveEnvPullTarget() err=%v", err)
	}
	if gotProject != root {
		t.Fatalf("project path = %q, want %q", gotProject, root)
	}
	wantFile := filepath.Join(root, ".env.local")
	if gotFile != wantFile {
		t.Fatalf("env file = %q, want %q", gotFile, wantFile)
	}
}

func TestCheckEnvPullTargetRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.env")
	if err := os.WriteFile(realFile, []byte("A = \"1\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() err=%v", err)
	}
	link := filepath.Join(dir, ".env.local")
	if err := os.Symlink(realFile, link); err != nil {
		t.Fatalf("Symlink() err=%v", err)
	}

	err := checkEnvPullTarget(link)
	assertValidationError(t, err, "must be a regular file")
}

func TestCheckEnvPullTargetRejectsDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".env.local")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() err=%v", err)
	}

	err := checkEnvPullTarget(dir)
	assertValidationError(t, err, "must be a regular file")
}

func TestParseEnvPullAssignmentLine(t *testing.T) {
	key, ok := parseEnvPullAssignmentLine("FOO = \"bar\"")
	if !ok {
		t.Fatalf("expected line to parse")
	}
	if key != "FOO" {
		t.Fatalf("key = %q, want FOO", key)
	}
}

func TestParseEnvPullAssignmentLineRejectsComment(t *testing.T) {
	if _, ok := parseEnvPullAssignmentLine("# FOO = \"bar\""); ok {
		t.Fatalf("commented line should not be treated as active assignment")
	}
}

func TestFormatEnvPullAssignmentEscapesQuotesAndBackslashes(t *testing.T) {
	got := formatEnvPullAssignment("TOKEN", `a"b\c`)
	want := `TOKEN = "a\"b\\c"`
	if got != want {
		t.Fatalf("formatEnvPullAssignment() = %q, want %q", got, want)
	}
}

func TestMergeEnvPullFileContentPreservesCommentsAndMalformedLines(t *testing.T) {
	original := strings.Join([]string{
		"# FOO = \"old\"",
		"FOO = \"old\"",
		"BROKEN LINE",
		"KEEP = \"stay\"",
		"",
	}, "\n")

	merged, updated, created := mergeEnvPullFileContent(original, map[string]string{
		"FOO": "new",
		"BAR": "added",
	})

	if !strings.Contains(merged, "# FOO = \"old\"") {
		t.Fatalf("comment line must be preserved: %q", merged)
	}
	if !strings.Contains(merged, "FOO = \"new\"") {
		t.Fatalf("active key must be updated: %q", merged)
	}
	if !strings.Contains(merged, "BROKEN LINE") {
		t.Fatalf("malformed line must be preserved: %q", merged)
	}
	if !strings.Contains(merged, "KEEP = \"stay\"") {
		t.Fatalf("unrelated key must be preserved: %q", merged)
	}
	if !strings.Contains(merged, "BAR = \"added\"") {
		t.Fatalf("missing key must be appended: %q", merged)
	}
	if len(updated) != 1 || updated[0] != "FOO" {
		t.Fatalf("updated = %v, want [FOO]", updated)
	}
	if len(created) != 1 || created[0] != "BAR" {
		t.Fatalf("created = %v, want [BAR]", created)
	}
}

func TestBuildEnvPullSuccessDataSuppressesEnvKeysAndValues(t *testing.T) {
	data := buildEnvPullSuccessData("app_x", "/repo", "/repo/.env.local", true, 1, 2)

	if _, ok := data["updated"]; ok {
		t.Fatalf("success data must not expose updated key names: %v", data)
	}
	if _, ok := data["created"]; ok {
		t.Fatalf("success data must not expose created key names: %v", data)
	}
	if data["updated_count"] != 1 {
		t.Fatalf("updated_count = %v, want 1", data["updated_count"])
	}
	if data["created_count"] != 2 {
		t.Fatalf("created_count = %v, want 2", data["created_count"])
	}
}

func TestAppsEnvPull_DryRunUsesPostAndResolvedEnvFile(t *testing.T) {
	factory, stdout, _ := newAppsExecuteFactory(t)
	projectDir := t.TempDir()

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--dry-run", "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("dry-run err=%v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, `"method": "POST"`) {
		t.Fatalf("dry-run must use POST: %s", got)
	}
	if !strings.Contains(got, `/open-apis/spark/v1/apps/app_x/env_vars`) {
		t.Fatalf("dry-run missing endpoint: %s", got)
	}
	if !strings.Contains(got, filepath.Join(projectDir, ".env.local")) {
		t.Fatalf("dry-run must include resolved env file path: %s", got)
	}
}

func TestAppsEnvPull_PrettyOutput_WithDatabaseLine(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"SUDA_DATABASE_URL": "short-lived-db-token",
					"APP_ID":            "app_x",
				},
			},
		},
	})

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--format", "pretty", "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "App detected: app_x") {
		t.Fatalf("missing app summary: %q", got)
	}
	if !strings.Contains(got, "Development database detected") {
		t.Fatalf("missing database line: %q", got)
	}
	if !strings.Contains(got, filepath.Join(projectDir, ".env.local")) {
		t.Fatalf("missing env file path in pretty output: %q", got)
	}
	if strings.Contains(got, "short-lived-db-token") {
		t.Fatalf("pretty output must not print env values: %q", got)
	}
}

func TestAppsEnvPull_JSONOutput_UsesCountsOnly(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"AAA": "value-a",
					"BBB": "value-b",
				},
			},
		},
	})

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, `"updated_count": 0`) {
		t.Fatalf("json output must expose updated_count: %s", got)
	}
	if !strings.Contains(got, `"created_count": 2`) {
		t.Fatalf("json output must expose created_count: %s", got)
	}
	if strings.Contains(got, `"AAA"`) || strings.Contains(got, `"BBB"`) {
		t.Fatalf("json output must not expose env key names: %s", got)
	}
	if strings.Contains(got, `"value-a"`) || strings.Contains(got, `"value-b"`) {
		t.Fatalf("json output must not expose env values: %s", got)
	}
}

func TestAppsEnvPull_MalformedPayloadFails(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": []interface{}{"bad"},
			},
		},
	})

	err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--as", "user"},
		factory, stdout)
	assertValidationError(t, err, "env_vars")
}

func TestAppsEnvPull_TargetSymlinkIsRejectedBeforeAPI(t *testing.T) {
	factory, stdout, _ := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	linkTarget := filepath.Join(projectDir, "real.env")
	if err := os.WriteFile(linkTarget, []byte("KEEP = \"1\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() err=%v", err)
	}
	if err := os.Symlink(linkTarget, filepath.Join(projectDir, ".env.local")); err != nil {
		t.Fatalf("Symlink() err=%v", err)
	}

	err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--as", "user"},
		factory, stdout)
	assertValidationError(t, err, "must be a regular file")
}

func TestReadEnvPullFile_MissingFileReturnsEmpty(t *testing.T) {
	got, err := readEnvPullFile(filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("readEnvPullFile() err=%v", err)
	}
	if got != "" {
		t.Fatalf("readEnvPullFile() = %q, want empty string", got)
	}
}

func TestAppsEnvPull_WritesCanonicalEnvFile(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"AAA": "new",
					"BBB": `quote"and\\slash`,
				},
			},
		},
	})
	if err := os.WriteFile(filepath.Join(projectDir, ".env.local"), []byte(strings.Join([]string{
		"# AAA = \"commented\"",
		"AAA = \"old\"",
		"KEEP = \"stay\"",
		"BROKEN LINE",
		"",
	}, "\n")), 0o600); err != nil {
		t.Fatalf("WriteFile() err=%v", err)
	}

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--format", "pretty", "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".env.local"))
	if err != nil {
		t.Fatalf("ReadFile() err=%v", err)
	}
	got := string(data)
	if !strings.Contains(got, "# AAA = \"commented\"") {
		t.Fatalf("comment must be preserved: %q", got)
	}
	if !strings.Contains(got, "AAA = \"new\"") {
		t.Fatalf("active value must be updated: %q", got)
	}
	if !strings.Contains(got, `BBB = "quote\"and\\\\slash"`) {
		t.Fatalf("new key must be appended canonically: %q", got)
	}
	if !strings.Contains(got, "KEEP = \"stay\"") {
		t.Fatalf("unrelated key must be preserved: %q", got)
	}
	if !strings.Contains(got, "BROKEN LINE") {
		t.Fatalf("malformed line must be preserved: %q", got)
	}
}

func TestAppsEnvPull_DryRunDoesNotWriteFile(t *testing.T) {
	factory, stdout, _ := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	target := filepath.Join(projectDir, ".env.local")

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--dry-run", "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("dry-run err=%v", err)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run must not create target file, stat err=%v", err)
	}
}

func TestAppsEnvPull_JSONOutputOmitsDatabaseLineText(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"SUDA_DATABASE_URL": "short-lived-db-token",
				},
			},
		},
	})

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}
	if strings.Contains(stdout.String(), "Development database detected") {
		t.Fatalf("json output must not include pretty text: %s", stdout.String())
	}
}

func TestAppsEnvPull_ValidationRequiresAppID(t *testing.T) {
	factory, stdout, _ := newAppsExecuteFactory(t)
	err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--project-path", t.TempDir(), "--as", "user"},
		factory, stdout)
	if err == nil || !strings.Contains(err.Error(), "app-id") {
		t.Fatalf("expected missing app-id error, got %v", err)
	}
}

func TestAppsEnvPull_ExecuteUsesNestedDataEnvVars(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"AAA": "value-a",
				},
			},
		},
	})

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(projectDir, ".env.local"))
	if err != nil {
		t.Fatalf("ReadFile() err=%v", err)
	}
	if !strings.Contains(string(data), `AAA = "value-a"`) {
		t.Fatalf("expected nested data env vars to be written, got %q", string(data))
	}
}

func TestAppsEnvPull_JSONOutputCanBeDecoded(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"AAA": "value-a",
				},
			},
		},
	})

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			AppID            string `json:"app_id"`
			ProjectPath      string `json:"project_path"`
			EnvFile          string `json:"env_file"`
			DatabaseDetected bool   `json:"database_detected"`
			UpdatedCount     int    `json:"updated_count"`
			CreatedCount     int    `json:"created_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() err=%v; stdout=%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true envelope, got %+v", envelope)
	}
	if envelope.Data.AppID != "app_x" {
		t.Fatalf("app_id = %q, want app_x", envelope.Data.AppID)
	}
}

func TestAppsEnvPull_PrettyOutputWithoutDatabaseLine(t *testing.T) {
	factory, stdout, reg := newAppsExecuteFactory(t)
	projectDir := t.TempDir()
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/spark/v1/apps/app_x/env_vars",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"env_vars": map[string]interface{}{
					"AAA": "value-a",
				},
			},
		},
	})

	if err := runAppsShortcut(t, AppsEnvPull,
		[]string{"+env-pull", "--app-id", "app_x", "--project-path", projectDir, "--format", "pretty", "--as", "user"},
		factory, stdout); err != nil {
		t.Fatalf("execute err=%v", err)
	}
	if strings.Contains(stdout.String(), "Development database detected") {
		t.Fatalf("unexpected database line in pretty output: %q", stdout.String())
	}
}

func TestMergeEnvPullFileContentEmptyEnvVarsPreservesOriginalNewline(t *testing.T) {
	original := "KEEP = \"stay\""
	merged, updated, created := mergeEnvPullFileContent(original, map[string]string{})
	if merged != "KEEP = \"stay\"\n" {
		t.Fatalf("merged = %q, want trailing newline preserved", merged)
	}
	if len(updated) != 0 || len(created) != 0 {
		t.Fatalf("updated=%v created=%v, want both empty", updated, created)
	}
}

func TestParseEnvPullAssignmentLineRejectsUnquotedValue(t *testing.T) {
	if _, ok := parseEnvPullAssignmentLine("FOO = bar"); ok {
		t.Fatalf("unquoted value should not be treated as active assignment")
	}
}

func TestResolveEnvPullTargetCleansCustomPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	input := filepath.Join(root, ".", "sub", "..")
	gotProject, gotFile, err := resolveEnvPullTarget(input)
	if err != nil {
		t.Fatalf("resolveEnvPullTarget() err=%v", err)
	}
	wantProject := filepath.Clean(input)
	if gotProject != wantProject {
		t.Fatalf("project path = %q, want %q", gotProject, wantProject)
	}
	if gotFile != filepath.Join(wantProject, ".env.local") {
		t.Fatalf("env file = %q, want %q", gotFile, filepath.Join(wantProject, ".env.local"))
	}
}

func TestWriteEnvPullPretty(t *testing.T) {
	var buf bytes.Buffer
	writeEnvPullPretty(&buf, "app_x", "/repo/.env.local", true)
	got := buf.String()
	if !strings.Contains(got, "App detected: app_x") {
		t.Fatalf("missing app line: %q", got)
	}
	if !strings.Contains(got, "Development database detected") {
		t.Fatalf("missing database line: %q", got)
	}
	if !strings.Contains(got, "/repo/.env.local") {
		t.Fatalf("missing env file path: %q", got)
	}
}

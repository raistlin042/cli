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
)

func TestAppsInit_Declaration(t *testing.T) {
	if AppsInit.Command != "+init" {
		t.Errorf("Command = %q, want +init", AppsInit.Command)
	}
	if AppsInit.Service != appsService {
		t.Errorf("Service = %q, want %q", AppsInit.Service, appsService)
	}
	if AppsInit.Risk != "write" {
		t.Errorf("Risk = %q, want write", AppsInit.Risk)
	}
	if !AppsInit.HasFormat {
		t.Error("HasFormat = false, want true")
	}
}

func TestDefaultCloneDir(t *testing.T) {
	got := defaultCloneDir("app_xyz")
	if got != filepath.Join(".", "app_xyz") {
		t.Errorf("defaultCloneDir = %q, want ./app_xyz", got)
	}
}

// --- pure-function tests ---

func TestParseRepoURL(t *testing.T) {
	url, err := parseRepoURLFromEnvelope(`{"ok":true,"data":{"repository_url":"http://u:t@h/app_x.git"}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://u:t@h/app_x.git" {
		t.Errorf("got %q", url)
	}
}

func TestParseRepoURL_Errors(t *testing.T) {
	for _, in := range []string{`not json`, `{"ok":false,"data":{}}`, `{"ok":true,"data":{}}`, `{"ok":true,"data":{"repository_url":""}}`} {
		if _, err := parseRepoURLFromEnvelope(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestValidateRepoURLScheme(t *testing.T) {
	for _, ok := range []string{"http://h/r.git", "https://h/r.git"} {
		if err := validateRepoURLScheme(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"ext::sh -c id", "file:///etc/passwd", "ssh://h/r", "-oProxyCommand=x", "git@h:r"} {
		if err := validateRepoURLScheme(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

// --- orchestration test helpers ---

func withFakeRunner(t *testing.T, f *fakeCommandRunner) {
	t.Helper()
	orig := initRunner
	initRunner = f
	t.Cleanup(func() { initRunner = orig })
}

func credInitOK(repoURL string) fakeCallResult {
	return fakeCallResult{stdout: `{"ok":true,"data":{"repository_url":"` + repoURL + `"}}`}
}

// relCloneDir returns a relative, cwd-contained, not-yet-existing directory
// name suitable for --dir. SafeInputPath rejects absolute paths (so
// t.TempDir() cannot be used directly) and requires the path stay under cwd.
// The fake runner never creates the dir, so ensureEmptyDir sees a missing path
// and passes. Cleanup removes it in case anything materializes it.
func relCloneDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	rel := "init-clone-" + strings.ReplaceAll(t.Name(), "/", "_")
	t.Cleanup(func() { os.RemoveAll(filepath.Join(cwd, rel)) })
	return rel
}

// parseEnvelopeData parses the JSON envelope's data object from stdout.
func parseEnvelopeData(t *testing.T, stdout *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (raw=%q)", err, stdout.String())
	}
	return env.Data
}

// findCall returns the recorded call whose name (element[1]) and first arg
// (element[2]) match, or nil if none.
func findCall(calls [][]string, name, firstArg string) []string {
	for _, c := range calls {
		if len(c) >= 3 && c[1] == name && c[2] == firstArg {
			return c
		}
	}
	return nil
}

// --- orchestration tests ---

func TestAppsInit_HappyPathCleanTree(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("http://u:t@h/app_x.git"),
		"git clone":       {},
		"git checkout":    {},
		"git status":      {},
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := parseEnvelopeData(t, stdout)
	if data["committed"] != false {
		t.Errorf("committed = %v, want false", data["committed"])
	}
	if data["pushed"] != false {
		t.Errorf("pushed = %v, want false", data["pushed"])
	}
	if data["npx_skipped"] != true {
		t.Errorf("npx_skipped = %v, want true", data["npx_skipped"])
	}
	if data["repository_url"] != "http://***@h/app_x.git" {
		t.Errorf("repository_url = %v, want redacted http://***@h/app_x.git", data["repository_url"])
	}
	clone := findCall(f.calls, "git", "clone")
	if clone == nil {
		t.Fatalf("git clone not recorded; calls=%v", f.calls)
	}
	// clone == [dir, "git", "clone", "--", repoURL, dir]; "--" must precede the URL.
	found := false
	for i := 0; i+1 < len(clone); i++ {
		if clone[i] == "--" && strings.HasPrefix(clone[i+1], "http") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("git clone args missing \"--\" immediately before URL: %v", clone)
	}
}

func TestAppsInit_DirtyTreeCommitPush(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("http://u:t@h/app_x.git"),
		"git clone":       {},
		"git checkout":    {},
		"git status":      {stdout: " M file.txt"},
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if findCall(f.calls, "git", "add") == nil {
		t.Errorf("git add not recorded; calls=%v", f.calls)
	}
	if findCall(f.calls, "git", "commit") == nil {
		t.Errorf("git commit not recorded; calls=%v", f.calls)
	}
	if findCall(f.calls, "git", "push") == nil {
		t.Errorf("git push not recorded; calls=%v", f.calls)
	}
	data := parseEnvelopeData(t, stdout)
	if data["committed"] != true {
		t.Errorf("committed = %v, want true", data["committed"])
	}
	if data["pushed"] != true {
		t.Errorf("pushed = %v, want true", data["pushed"])
	}
}

func TestAppsInit_CredentialInitFailure(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": {stderr: "boom", err: errors.New("exit 1")},
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), ":t@") {
		t.Errorf("error leaks token: %v", err)
	}
}

func TestAppsInit_BadRepoURLScheme(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("ext::sh -c id"),
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if findCall(f.calls, "git", "clone") != nil {
		t.Errorf("git clone should not be recorded for bad scheme; calls=%v", f.calls)
	}
}

func TestAppsInit_CloneFailure(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("http://u:t@h/r.git"),
		"git clone":       {stderr: "fatal: unable to access 'http://u:t@h/r.git'", err: errors.New("exit 128")},
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), "u:t@") {
		t.Errorf("error leaks credentials: %v", err)
	}
	if !strings.Contains(err.Error(), "***") {
		t.Errorf("error should be redacted with ***: %v", err)
	}
}

func TestAppsInit_PushFailure(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("http://u:t@h/app_x.git"),
		"git clone":       {},
		"git checkout":    {},
		"git status":      {stdout: " M file.txt"},
		"git push":        {err: errors.New("exit 1")},
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestAppsInit_DirNonEmpty(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("http://u:t@h/app_x.git"),
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)

	// Create a non-empty directory under cwd (SafeInputPath requires relative,
	// cwd-contained paths), then pass it as --dir.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	nonEmpty, err := os.MkdirTemp(cwd, "init-nonempty-")
	if err != nil {
		t.Fatalf("mkdirtemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(nonEmpty) })
	if err := os.WriteFile(filepath.Join(nonEmpty, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err = runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", filepath.Base(nonEmpty), "--as", "user"}, factory, stdout)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	if len(f.calls) != 0 {
		t.Errorf("no runner calls expected before dir rejection; calls=%v", f.calls)
	}
}

func TestAppsInit_AsPassthrough(t *testing.T) {
	f := &fakeCommandRunner{results: map[string]fakeCallResult{
		"credential-init": credInitOK("http://u:t@h/app_x.git"),
		"git clone":       {},
		"git checkout":    {},
		"git status":      {},
	}}
	withFakeRunner(t, f)
	factory, stdout, _ := newAppsExecuteFactory(t)
	dir := relCloneDir(t)

	// AppsInit.AuthTypes is ["user"], so the framework rejects --as bot. Use
	// --as user and assert it is forwarded to the self-invoked credential-init.
	err := runAppsShortcut(t, AppsInit, []string{"+init", "--app-id", "app_x", "--dir", dir, "--as", "user"}, factory, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var cred []string
	for _, c := range f.calls {
		if len(c) >= 3 && c[2] == "apps" {
			cred = c
			break
		}
	}
	if cred == nil {
		t.Fatalf("credential-init call not recorded; calls=%v", f.calls)
	}
	hasAs, hasUser := false, false
	for _, a := range cred {
		if a == "--as" {
			hasAs = true
		}
		if a == "user" {
			hasUser = true
		}
	}
	if !hasAs || !hasUser {
		t.Errorf("credential-init args missing --as user: %v", cred)
	}
}

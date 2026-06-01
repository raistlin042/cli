// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"path/filepath"
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

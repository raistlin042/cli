// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

// 钉死域内 shortcut 数量。少一条（漏挂）或多一条（误加）都会被这个测试拦截。
func TestAppsShortcuts_Returns11(t *testing.T) {
	got := Shortcuts()
	if len(got) != 11 {
		t.Fatalf("Shortcuts() returned %d entries, want 11", len(got))
	}
}

// 确认 5 个 session 生命周期命令都已挂载。
func TestAppsShortcuts_IncludesSessionCommands(t *testing.T) {
	want := map[string]bool{
		"+session-create": false,
		"+session-list":   false,
		"+session-read":   false,
		"+session-stop":   false,
		"+chat":           false,
	}
	for _, sc := range Shortcuts() {
		if _, ok := want[sc.Command]; ok {
			want[sc.Command] = true
		}
	}
	for cmd, found := range want {
		if !found {
			t.Errorf("Shortcuts() missing %s", cmd)
		}
	}
}

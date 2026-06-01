// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestBuildHistoryBody(t *testing.T) {
	b := buildHistoryBody("app_x", 0, "")
	if _, ok := b["limit"]; ok {
		t.Errorf("limit should be omitted when 0, got %v", b)
	}
	if _, ok := b["pageToken"]; ok {
		t.Errorf("pageToken should be omitted when empty, got %v", b)
	}
	b2 := buildHistoryBody("app_x", 30, "tok")
	if b2["limit"] != 30 || b2["pageToken"] != "tok" {
		t.Errorf("body = %v", b2)
	}
}

func TestValidateHistoryLimit(t *testing.T) {
	if err := validateHistoryLimit(0); err != nil {
		t.Errorf("limit 0 (unset) should pass, got %v", err)
	}
	if err := validateHistoryLimit(500); err != nil {
		t.Errorf("limit 500 should pass, got %v", err)
	}
	if err := validateHistoryLimit(501); err == nil {
		t.Error("limit 501 should fail")
	}
	if err := validateHistoryLimit(-1); err == nil {
		t.Error("limit -1 should fail")
	}
}

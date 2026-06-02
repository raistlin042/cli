// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestBuildHistoryQuery(t *testing.T) {
	// status, limit, page_token omitted when zero/empty; app_id is in the path
	q := buildHistoryQuery("", 0, "")
	if _, ok := q["status"]; ok {
		t.Errorf("status should be omitted when empty, got %v", q)
	}
	if _, ok := q["limit"]; ok {
		t.Errorf("limit should be omitted when 0, got %v", q)
	}
	if _, ok := q["page_token"]; ok {
		t.Errorf("page_token should be omitted when empty, got %v", q)
	}
	// all params included; page_token uses snake_case key
	q2 := buildHistoryQuery("finished", 30, "tok")
	if q2["status"] != "finished" {
		t.Errorf("status = %v, want finished", q2["status"])
	}
	if q2["limit"] != 30 {
		t.Errorf("limit = %v, want 30", q2["limit"])
	}
	if q2["page_token"] != "tok" {
		t.Errorf("page_token = %v, want tok", q2["page_token"])
	}
	if _, ok := q2["app_id"]; ok {
		t.Errorf("app_id must not be in query params, got %v", q2)
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

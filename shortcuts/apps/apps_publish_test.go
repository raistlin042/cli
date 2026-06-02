// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestBuildPublishBody(t *testing.T) {
	// branch included when non-empty; app_id is NOT in body (it's in the path)
	b := buildPublishBody("feat/devops")
	if b["branch"] != "feat/devops" {
		t.Errorf("body = %v", b)
	}
	if _, ok := b["app_id"]; ok {
		t.Errorf("app_id must not be in body, got %v", b)
	}
	// branch omitted when empty
	b2 := buildPublishBody("")
	if _, ok := b2["branch"]; ok {
		t.Errorf("branch should be omitted when empty, got %v", b2)
	}
}

func TestAppsPublishMeta(t *testing.T) {
	if AppsPublish.Command != "+publish" || AppsPublish.Risk != "write" {
		t.Errorf("meta mismatch: %+v", AppsPublish)
	}
	if len(AppsPublish.Scopes) != 1 || AppsPublish.Scopes[0] != "spark:app:write" {
		t.Errorf("scopes = %v", AppsPublish.Scopes)
	}
}

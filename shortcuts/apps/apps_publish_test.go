// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestBuildPublishBody(t *testing.T) {
	b := buildPublishBody("app_x", "feat/devops")
	if b["appID"] != "app_x" || b["branch"] != "feat/devops" {
		t.Errorf("body = %v", b)
	}
	b2 := buildPublishBody("app_x", "")
	if _, ok := b2["branch"]; ok {
		t.Errorf("branch should be omitted, got %v", b2)
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

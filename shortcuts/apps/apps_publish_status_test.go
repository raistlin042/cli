// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestAppsPublishStatusMeta(t *testing.T) {
	if AppsPublishStatus.Command != "+publish-status" || AppsPublishStatus.Risk != "read" {
		t.Errorf("meta mismatch: %+v", AppsPublishStatus)
	}
	if len(AppsPublishStatus.Scopes) != 1 || AppsPublishStatus.Scopes[0] != "spark:app:read" {
		t.Errorf("scopes = %v", AppsPublishStatus.Scopes)
	}
	// both --app-id and --instance-id must be required
	req := map[string]bool{}
	for _, f := range AppsPublishStatus.Flags {
		req[f.Name] = f.Required
	}
	if !req["app-id"] || !req["instance-id"] {
		t.Errorf("app-id and instance-id must be Required; flags=%+v", AppsPublishStatus.Flags)
	}
}

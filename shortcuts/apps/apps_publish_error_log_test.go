// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestShapeErrorLog(t *testing.T) {
	in := map[string]interface{}{
		"status": "failed",
		"errorLogs": []interface{}{
			map[string]interface{}{"step": "build", "errorLog": "boom"},
		},
	}
	out := shapeErrorLog(in)
	if out["status"] != "failed" {
		t.Errorf("status = %v, want failed", out["status"])
	}
	logs, ok := out["error_logs"].([]interface{})
	if !ok || len(logs) != 1 {
		t.Fatalf("error_logs = %v", out["error_logs"])
	}
	// missing errorLogs -> empty slice, not nil
	out2 := shapeErrorLog(map[string]interface{}{"status": "finished"})
	if logs2, ok := out2["error_logs"].([]interface{}); !ok || len(logs2) != 0 {
		t.Errorf("error_logs should default to empty slice, got %v", out2["error_logs"])
	}
}

func TestAppsPublishErrorLogMeta(t *testing.T) {
	if AppsPublishErrorLog.Command != "+publish-error-log" || AppsPublishErrorLog.Risk != "read" {
		t.Errorf("meta mismatch: %+v", AppsPublishErrorLog)
	}
	if len(AppsPublishErrorLog.Scopes) != 1 || AppsPublishErrorLog.Scopes[0] != "spark:app:read" {
		t.Errorf("scopes = %v", AppsPublishErrorLog.Scopes)
	}
	req := map[string]bool{}
	for _, f := range AppsPublishErrorLog.Flags {
		req[f.Name] = f.Required
	}
	if !req["app-id"] || !req["release-id"] {
		t.Errorf("app-id and release-id must be Required; flags=%+v", AppsPublishErrorLog.Flags)
	}
}

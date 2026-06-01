// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestShapeErrorLog(t *testing.T) {
	in := map[string]interface{}{
		"status": float64(4),
		"errorJobs": []interface{}{
			map[string]interface{}{"jobID": "j1", "componentName": "c1", "errorMsg": "boom"},
		},
	}
	out := shapeErrorLog(in)
	if out["status_name"] != "Failed" {
		t.Errorf("status_name = %v, want Failed", out["status_name"])
	}
	jobs, ok := out["error_jobs"].([]interface{})
	if !ok || len(jobs) != 1 {
		t.Fatalf("error_jobs = %v", out["error_jobs"])
	}
	// missing errorJobs -> empty slice, not nil
	out2 := shapeErrorLog(map[string]interface{}{"status": float64(3)})
	if jobs2, ok := out2["error_jobs"].([]interface{}); !ok || len(jobs2) != 0 {
		t.Errorf("error_jobs should default to empty slice, got %v", out2["error_jobs"])
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
	if !req["app-id"] || !req["instance-id"] {
		t.Errorf("app-id and instance-id must be Required; flags=%+v", AppsPublishErrorLog.Flags)
	}
}

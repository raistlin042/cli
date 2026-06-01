// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"
	"fmt"

	"github.com/larksuite/cli/internal/output"
)

// publishAPIWired reports whether the devops/pipeline publish endpoints have
// been deployed to the OpenAPI gateway. While false, each command's Execute
// returns a structured "unavailable" error and only --dry-run works.
//
// TODO(apps-publish): once lark.apaas.devops / lark.apaas.devops_platform are
// exposed on the OpenAPI gateway, fill in the four gateway paths below and set
// publishAPIWired = true. The runtime guard (ensurePublishWired) deactivates
// automatically once this flips.
const publishAPIWired = false

// TODO(apps-publish): replace with the real OpenAPI gateway paths once known.
// Left empty on purpose — do NOT fabricate gateway addresses. These are only
// referenced by Execute, which never runs while publishAPIWired == false.
// Upstream BAM references (NOT gateway paths):
//
//	create     POST /v1/devops/app/:appID/release                  (endpoint 4070318)
//	history    POST /v1/pipeline/app/:appID/instance/list          (endpoint 4073969)
//	status     GET  /v1/pipeline/app/:appID/instance/:id           (endpoint 4073971)
//	error-log  GET  /v1/pipeline/app/:appID/instance/:id/error_log (endpoint 4073972)
//
// Declared as var (not const) so go vet's printf analyzer does not flag the
// fmt.Sprintf calls in Execute while these are empty TODO placeholders. Once a
// real "/...%s..." gateway path is filled in (and publishAPIWired flips), the
// fmt.Sprintf calls become exactly correct. See apps_publish_common.go header.
var (
	publishCreatePath   = ""
	publishHistoryPath  = ""
	publishStatusPath   = ""
	publishErrorLogPath = ""
)

// Upstream PSM reference paths shown in --dry-run output. These are the
// documented upstream paths (from BAM), explicitly NOT gateway paths — dry-run
// labels them as such via each command's Desc and gateway_status field.
const (
	upstreamCreatePath   = "/v1/devops/app/%s/release"
	upstreamHistoryPath  = "/v1/pipeline/app/%s/instance/list"
	upstreamStatusPath   = "/v1/pipeline/app/%s/instance/%s"
	upstreamErrorLogPath = "/v1/pipeline/app/%s/instance/%s/error_log"
)

// ensurePublishWired is the Execute-time guard. While the endpoints are not on
// the OpenAPI gateway it returns a structured error so callers get a clear
// message instead of a confusing low-level HTTP failure.
func ensurePublishWired() error {
	if publishAPIWired {
		return nil
	}
	return output.ErrWithHint(output.ExitAPI, "unavailable",
		"apps publish endpoints are not yet deployed to the OpenAPI gateway",
		"only --dry-run is available for now; once lark.apaas.devops / lark.apaas.devops_platform are exposed, fill the gateway paths in apps_publish_common.go and set publishAPIWired=true")
}

// nodeStatusName maps the upstream NodeStatus enum to a human-readable name.
// Mirrors devops_platform/common/common.NodeStatus (BAM v1.0.293).
func nodeStatusName(n int) string {
	switch n {
	case 0:
		return "Unspecified"
	case 1:
		return "ToDo"
	case 2:
		return "Running"
	case 3:
		return "Success"
	case 4:
		return "Failed"
	case 5:
		return "Canceled"
	case 6:
		return "HoldOn"
	default:
		return fmt.Sprintf("Unknown(%d)", n)
	}
}

// toInt coerces a JSON-decoded numeric value (float64 / json.Number / int) to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

// injectStatusName adds a "status_name" field next to a numeric "status" field.
// No-op when m is nil or has no "status" key.
func injectStatusName(m map[string]interface{}) {
	if m == nil {
		return
	}
	if s, ok := m["status"]; ok {
		m["status_name"] = nodeStatusName(toInt(s))
	}
}

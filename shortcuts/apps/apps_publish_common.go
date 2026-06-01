// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"
	"fmt"

	"github.com/larksuite/cli/internal/output"
)

// publishAPIWired reports whether the devops publish endpoints have been
// deployed to the OpenAPI gateway. While false, each command's Execute returns
// a structured "unavailable" error and only --dry-run works.
//
// TODO(apps-publish): once lark.apaas.devops v1.0.381 RPC methods are exposed
// on the OpenAPI gateway, fill in the four gateway paths below and set
// publishAPIWired = true. The runtime guard (ensurePublishWired) deactivates
// automatically once this flips.
const publishAPIWired = false

// TODO(apps-publish): replace with the real OpenAPI gateway paths once known.
// Left empty on purpose — do NOT fabricate gateway addresses. These are only
// referenced by Execute, which never runs while publishAPIWired == false.
// Upstream RPC references (PSM lark.apaas.devops v1.0.381, NOT gateway paths):
//
//	create     rpc OpenAPICreateRelease    (endpoint 4177527)
//	list       rpc OpenAPIListReleases     (endpoint 4177529)
//	get        rpc OpenAPIGetRelease       (endpoint 4177526)
//	error-log  rpc OpenAPIGetReleaseErrorLogs (endpoint 4177528)
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

// RPC method names for lark.apaas.devops v1.0.381.
// These are the upstream RPC method names shown in --dry-run output.
const (
	rpcCreateRelease       = "OpenAPICreateRelease"
	rpcGetRelease          = "OpenAPIGetRelease"
	rpcListReleases        = "OpenAPIListReleases"
	rpcGetReleaseErrorLogs = "OpenAPIGetReleaseErrorLogs"
)

// ensurePublishWired is the Execute-time guard. While the endpoints are not on
// the OpenAPI gateway it returns a structured error so callers get a clear
// message instead of a confusing low-level HTTP failure.
func ensurePublishWired() error {
	if publishAPIWired {
		return nil
	}
	// User-facing hint stays in user language; the wiring detail (fill the
	// gateway paths + flip publishAPIWired) lives in the publishAPIWired comment
	// block above, where the maintainer who enables it will be looking.
	return output.ErrWithHint(output.ExitAPI, "unavailable",
		"apps publish endpoints are not yet deployed to the OpenAPI gateway",
		"this feature is not available yet — use --dry-run to preview the request; it will be enabled once the endpoint is deployed")
}

// releaseStatusName maps the upstream ReleaseStatus enum to a human-readable name.
// Mirrors lark.apaas.devops ReleaseStatus (v1.0.381).
func releaseStatusName(n int) string {
	switch n {
	case 0:
		return "Unspecified"
	case 1:
		return "Publishing"
	case 2:
		return "Finished"
	case 3:
		return "Failed"
	case 4:
		return "Canceled"
	case 5:
		return "Rollback"
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
		m["status_name"] = releaseStatusName(toInt(s))
	}
}

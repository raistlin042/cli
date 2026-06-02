// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"github.com/larksuite/cli/internal/output"
)

// publishAPIWired reports whether the devops publish endpoints have been
// deployed to the OpenAPI gateway. While false, each command's Execute returns
// a structured "unavailable" error and only --dry-run works.
//
// Real gateway paths are known (see consts below). publishAPIWired stays false
// until the endpoint is deployed. Flip to true is a 1-line change; Execute path
// + request/response shapes are already correct against the final gateway def.
const publishAPIWired = false

// Real OpenAPI gateway paths for the apps publish endpoints.
// Prefix reuses apiBasePath = "/open-apis/spark/v1" (same package).
// Each path contains %s placeholders; use fmt.Sprintf to build the final URL.
const (
	publishCreatePath   = apiBasePath + "/apps/%s/releases"
	publishGetPath      = apiBasePath + "/apps/%s/releases/%s"
	publishErrorLogPath = apiBasePath + "/apps/%s/releases/%s/error_logs"
	publishListPath     = apiBasePath + "/apps/%s/releases"
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
		"this feature is not available yet — use --dry-run to preview the request; it will be enabled once the endpoint is deployed")
}

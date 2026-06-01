// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

// AppsPublishErrorLog fetches the error log for a release.
//
// NOTE: upstream endpoint (lark.apaas.devops v1.0.381, rpc OpenAPIGetReleaseErrorLogs,
// endpoint 4177528) not yet on the OpenAPI gateway; Execute gated by
// ensurePublishWired(). See apps_publish_common.go.
var AppsPublishErrorLog = common.Shortcut{
	Service:     appsService,
	Command:     "+publish-error-log",
	Description: "Get the error log for a release",
	Risk:        "read",
	Scopes:      []string{"spark:app:read"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "Miaoda app ID", Required: true},
		{Name: "release-id", Desc: "release ID (the release_id returned by +publish)", Required: true},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		if strings.TrimSpace(rctx.Str("release-id")) == "" {
			return output.ErrValidation("--release-id is required")
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		releaseID := strings.TrimSpace(rctx.Str("release-id"))
		dry := common.NewDryRunAPI()
		dry.Desc("Get release error log — NOT YET ON OPENAPI GATEWAY (rpc reference shown; real call returns 'unavailable' until publishAPIWired=true)")
		dry.Set("psm", "lark.apaas.devops")
		dry.Set("rpc_method", rpcGetReleaseErrorLogs)
		dry.Set("request", map[string]interface{}{"appID": appID, "releaseID": releaseID})
		dry.Set("gateway_status", "not_deployed")
		dry.Set("note", "endpoint not yet on OpenAPI gateway; rpc_method is the upstream reference, not a gateway path")
		return dry
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if err := ensurePublishWired(); err != nil {
			return err
		}
		appID := strings.TrimSpace(rctx.Str("app-id"))
		releaseID := strings.TrimSpace(rctx.Str("release-id"))
		path := fmt.Sprintf(publishErrorLogPath, validate.EncodePathSegment(appID), validate.EncodePathSegment(releaseID))
		data, err := rctx.CallAPI("GET", path, map[string]interface{}{"appID": appID, "releaseID": releaseID}, nil)
		if err != nil {
			return err
		}
		out := shapeErrorLog(data)
		rctx.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "status: %v\n", out["status"])
			logs, _ := out["error_logs"].([]interface{})
			rows := make([]map[string]interface{}, 0, len(logs))
			for _, l := range logs {
				m, ok := l.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, map[string]interface{}{
					"step":     m["step"],
					"errorLog": m["errorLog"],
				})
			}
			output.PrintTable(w, rows)
		})
		return nil
	},
}

// shapeErrorLog maps the upstream error-log response into the CLI envelope:
// status is a string passthrough; normalises errorLogs -> error_logs.
func shapeErrorLog(data map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{"status": data["status"]}
	if logs, ok := data["errorLogs"]; ok {
		out["error_logs"] = logs
	} else {
		out["error_logs"] = []interface{}{}
	}
	return out
}

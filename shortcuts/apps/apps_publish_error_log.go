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

// AppsPublishErrorLog fetches the error log (failed jobs) for a release instance.
//
// NOTE: upstream endpoint (BAM 4073972) not yet on the OpenAPI gateway; Execute
// gated by ensurePublishWired(). See apps_publish_common.go.
var AppsPublishErrorLog = common.Shortcut{
	Service:     appsService,
	Command:     "+publish-error-log",
	Description: "Get the error log (failed jobs) for a release instance",
	Risk:        "read",
	Scopes:      []string{"spark:app:read"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "Miaoda app ID", Required: true},
		{Name: "instance-id", Desc: "release instance ID (the instance_id returned by +publish)", Required: true},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		if strings.TrimSpace(rctx.Str("instance-id")) == "" {
			return output.ErrValidation("--instance-id is required")
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		instanceID := strings.TrimSpace(rctx.Str("instance-id"))
		dry := common.NewDryRunAPI()
		dry.GET(fmt.Sprintf(upstreamErrorLogPath, validate.EncodePathSegment(appID), validate.EncodePathSegment(instanceID))).
			Desc("Get release error log — NOT YET ON OPENAPI GATEWAY (upstream PSM path shown; real call returns 'unavailable' until publishAPIWired=true)").
			Params(map[string]interface{}{"instanceID": instanceID})
		dry.Set("gateway_status", "not_deployed")
		dry.Set("note", "endpoint not yet on OpenAPI gateway; url is the upstream PSM reference, not a gateway path")
		return dry
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if err := ensurePublishWired(); err != nil {
			return err
		}
		appID := strings.TrimSpace(rctx.Str("app-id"))
		instanceID := strings.TrimSpace(rctx.Str("instance-id"))
		path := fmt.Sprintf(publishErrorLogPath, validate.EncodePathSegment(appID), validate.EncodePathSegment(instanceID))
		params := map[string]interface{}{"instanceID": instanceID}
		data, err := rctx.CallAPI("GET", path, params, nil)
		if err != nil {
			return err
		}
		out := shapeErrorLog(data)
		rctx.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "status: %v\n", out["status_name"])
			jobs, _ := out["error_jobs"].([]interface{})
			rows := make([]map[string]interface{}, 0, len(jobs))
			for _, j := range jobs {
				m, ok := j.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, map[string]interface{}{
					"jobID":         m["jobID"],
					"componentName": m["componentName"],
					"errorMsg":      m["errorMsg"],
				})
			}
			output.PrintTable(w, rows)
		})
		return nil
	},
}

// shapeErrorLog maps the upstream error-log response into the CLI envelope:
// adds status_name and renames errorJobs -> error_jobs.
func shapeErrorLog(data map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{
		"status":      data["status"],
		"status_name": nodeStatusName(toInt(data["status"])),
	}
	if jobs, ok := data["errorJobs"]; ok {
		out["error_jobs"] = jobs
	} else {
		out["error_jobs"] = []interface{}{}
	}
	return out
}

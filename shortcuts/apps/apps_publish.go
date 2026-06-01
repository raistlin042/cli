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

// AppsPublish creates a release for a Miaoda app.
//
// NOTE: the upstream endpoint (BAM 4070318, lark.apaas.devops) is not yet on
// the OpenAPI gateway. Execute is gated by ensurePublishWired(); only --dry-run
// works until publishAPIWired flips. See apps_publish_common.go.
var AppsPublish = common.Shortcut{
	Service:     appsService,
	Command:     "+publish",
	Description: "Create a release for a Miaoda app (returns instance_id for status polling)",
	Risk:        "write",
	Scopes:      []string{"spark:app:write"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "Miaoda app ID", Required: true},
		{Name: "branch", Desc: "release branch (server uses default if omitted)"},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		branch := strings.TrimSpace(rctx.Str("branch"))
		dry := common.NewDryRunAPI()
		dry.POST(fmt.Sprintf(upstreamCreatePath, validate.EncodePathSegment(appID))).
			Desc("Create release — NOT YET ON OPENAPI GATEWAY (upstream PSM path shown; real call returns 'unavailable' until publishAPIWired=true)").
			Body(buildPublishBody(appID, branch))
		dry.Set("gateway_status", "not_deployed")
		dry.Set("note", "endpoint not yet on OpenAPI gateway; url is the upstream PSM reference, not a gateway path")
		return dry
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if err := ensurePublishWired(); err != nil {
			return err
		}
		appID := strings.TrimSpace(rctx.Str("app-id"))
		branch := strings.TrimSpace(rctx.Str("branch"))
		path := strings.Replace(publishCreatePath, "%s", validate.EncodePathSegment(appID), 1)
		data, err := rctx.CallAPI("POST", path, nil, buildPublishBody(appID, branch))
		if err != nil {
			return err
		}
		taskID := common.GetString(data, "pipelineTaskID")
		out := map[string]interface{}{
			"instance_id":      taskID,
			"pipeline_task_id": taskID,
		}
		rctx.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "instance_id: %s\n", taskID)
		})
		return nil
	},
}

// buildPublishBody builds the create-release request body. branch is omitted
// when empty so the server applies its default.
func buildPublishBody(appID, branch string) map[string]interface{} {
	body := map[string]interface{}{"appID": appID}
	if branch != "" {
		body["branch"] = branch
	}
	return body
}

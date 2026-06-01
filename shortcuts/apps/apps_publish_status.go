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

// AppsPublishStatus fetches a single release instance's detail by instance ID.
//
// NOTE: upstream endpoint (BAM 4073971) not yet on the OpenAPI gateway; Execute
// gated by ensurePublishWired(). See apps_publish_common.go.
var AppsPublishStatus = common.Shortcut{
	Service:     appsService,
	Command:     "+publish-status",
	Description: "Get a single release's status/detail by instance ID",
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
		dry.GET(fmt.Sprintf(upstreamStatusPath, validate.EncodePathSegment(appID), validate.EncodePathSegment(instanceID))).
			Desc("Get release detail — NOT YET ON OPENAPI GATEWAY (upstream PSM path shown; real call returns 'unavailable' until publishAPIWired=true)").
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
		path := fmt.Sprintf(publishStatusPath, validate.EncodePathSegment(appID), validate.EncodePathSegment(instanceID))
		params := map[string]interface{}{"instanceID": instanceID}
		data, err := rctx.CallAPI("GET", path, params, nil)
		if err != nil {
			return err
		}
		out := data
		if instance, ok := data["instance"].(map[string]interface{}); ok {
			injectStatusName(instance)
			out = instance
		}
		rctx.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "ID: %v\nstatus: %v\nappID: %v\ncreator: %v\ncreatedAt: %v\nupdatedAt: %v\n",
				out["ID"], out["status_name"], out["appID"], out["creator"], out["createdAt"], out["updatedAt"])
		})
		return nil
	},
}

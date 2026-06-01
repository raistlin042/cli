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

// AppsPublishStatus fetches a single release's detail by release ID.
//
// NOTE: upstream endpoint (lark.apaas.devops v1.0.381, rpc OpenAPIGetRelease,
// endpoint 4177526) not yet on the OpenAPI gateway; Execute gated by
// ensurePublishWired(). See apps_publish_common.go.
var AppsPublishStatus = common.Shortcut{
	Service:     appsService,
	Command:     "+publish-status",
	Description: "Get a single release's status/detail by release ID",
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
		dry.Desc("Get release detail — NOT YET ON OPENAPI GATEWAY (rpc reference shown; real call returns 'unavailable' until publishAPIWired=true)")
		dry.Set("psm", "lark.apaas.devops")
		dry.Set("rpc_method", rpcGetRelease)
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
		path := fmt.Sprintf(publishStatusPath, validate.EncodePathSegment(appID), validate.EncodePathSegment(releaseID))
		data, err := rctx.CallAPI("GET", path, map[string]interface{}{"appID": appID, "releaseID": releaseID}, nil)
		if err != nil {
			return err
		}
		out := data
		if release, ok := data["release"].(map[string]interface{}); ok {
			out = release
		}
		rctx.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "releaseID: %v\nstatus: %v\ncreatedAt: %v\nupdatedAt: %v\n",
				out["releaseID"], out["status"], out["createdAt"], out["updatedAt"])
		})
		return nil
	},
}

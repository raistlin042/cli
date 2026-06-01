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
// NOTE: the upstream endpoint (lark.apaas.devops v1.0.381, rpc OpenAPICreateRelease,
// endpoint 4177527) is not yet on the OpenAPI gateway. Execute is gated by
// ensurePublishWired(); only --dry-run works until publishAPIWired flips.
// See apps_publish_common.go.
var AppsPublish = common.Shortcut{
	Service:     appsService,
	Command:     "+publish",
	Description: "Create a release for a Miaoda app (returns release_id for status polling)",
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
		dry.Desc("Create release — NOT YET ON OPENAPI GATEWAY (rpc reference shown; real call returns 'unavailable' until publishAPIWired=true)")
		dry.Set("psm", "lark.apaas.devops")
		dry.Set("rpc_method", rpcCreateRelease)
		dry.Set("request", buildPublishBody(appID, branch))
		dry.Set("gateway_status", "not_deployed")
		dry.Set("note", "endpoint not yet on OpenAPI gateway; rpc_method is the upstream reference, not a gateway path")
		return dry
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if err := ensurePublishWired(); err != nil {
			return err
		}
		appID := strings.TrimSpace(rctx.Str("app-id"))
		branch := strings.TrimSpace(rctx.Str("branch"))
		path := fmt.Sprintf(publishCreatePath, validate.EncodePathSegment(appID))
		data, err := rctx.CallAPI("POST", path, nil, buildPublishBody(appID, branch))
		if err != nil {
			return err
		}
		releaseID := common.GetString(data, "releaseID")
		status := common.GetString(data, "status")
		out := map[string]interface{}{
			"release_id": releaseID,
			"status":     status,
		}
		rctx.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "release_id: %s\nstatus: %s\n", releaseID, status)
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

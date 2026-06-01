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

// AppsPublishHistory lists a Miaoda app's release history (most recent first).
//
// NOTE: upstream endpoint (lark.apaas.devops v1.0.381, rpc OpenAPIListReleases,
// endpoint 4177529) not yet on the OpenAPI gateway; Execute gated by
// ensurePublishWired(). See apps_publish_common.go.
var AppsPublishHistory = common.Shortcut{
	Service:     appsService,
	Command:     "+publish-history",
	Description: "List a Miaoda app's release history (server returns ~50 most recent by default)",
	Risk:        "read",
	Scopes:      []string{"spark:app:read"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "Miaoda app ID", Required: true},
		{Name: "status", Enum: []string{"publishing", "finished", "failed"}, Desc: "filter by release status: publishing | finished | failed"},
		{Name: "limit", Type: "int", Desc: "page size (1-500); omit to use server default (~50)"},
		{Name: "page-token", Desc: "pagination cursor from a previous response"},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		return validateHistoryLimit(rctx.Int("limit"))
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		dry := common.NewDryRunAPI()
		dry.Desc("List release history — NOT YET ON OPENAPI GATEWAY (rpc reference shown; real call returns 'unavailable' until publishAPIWired=true)")
		dry.Set("psm", "lark.apaas.devops")
		dry.Set("rpc_method", rpcListReleases)
		dry.Set("request", buildHistoryBody(appID, strings.TrimSpace(rctx.Str("status")), rctx.Int("limit"), strings.TrimSpace(rctx.Str("page-token"))))
		dry.Set("gateway_status", "not_deployed")
		dry.Set("note", "endpoint not yet on OpenAPI gateway; rpc_method is the upstream reference, not a gateway path")
		return dry
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if err := ensurePublishWired(); err != nil {
			return err
		}
		appID := strings.TrimSpace(rctx.Str("app-id"))
		path := fmt.Sprintf(publishHistoryPath, validate.EncodePathSegment(appID))
		body := buildHistoryBody(appID, strings.TrimSpace(rctx.Str("status")), rctx.Int("limit"), strings.TrimSpace(rctx.Str("page-token")))
		data, err := rctx.CallAPI("POST", path, nil, body)
		if err != nil {
			return err
		}
		releases, _ := data["releases"].([]interface{})
		rctx.OutFormat(data, nil, func(w io.Writer) {
			rows := make([]map[string]interface{}, 0, len(releases))
			for _, it := range releases {
				m, ok := it.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, map[string]interface{}{
					"releaseID": m["releaseID"],
					"status":    m["status"],
					"createdAt": m["createdAt"],
					"updatedAt": m["updatedAt"],
				})
			}
			output.PrintTable(w, rows)
		})
		return nil
	},
}

// buildHistoryBody builds the list-releases body. status is sent only when non-empty;
// limit only when > 0 (0 means "not provided"); pageToken only when non-empty.
func buildHistoryBody(appID string, status string, limit int, pageToken string) map[string]interface{} {
	body := map[string]interface{}{"appID": appID}
	if status != "" {
		body["status"] = status
	}
	if limit > 0 {
		body["limit"] = limit
	}
	if pageToken != "" {
		body["pageToken"] = pageToken
	}
	return body
}

// validateHistoryLimit accepts 0 (unset) or 1-500.
func validateHistoryLimit(limit int) error {
	if limit == 0 {
		return nil
	}
	if limit < 1 || limit > 500 {
		return output.ErrValidation("--limit must be between 1 and 500")
	}
	return nil
}

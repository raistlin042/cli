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
// NOTE: upstream endpoint (BAM 4073969) not yet on the OpenAPI gateway; Execute
// gated by ensurePublishWired(). See apps_publish_common.go.
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
		dry.POST(fmt.Sprintf(upstreamHistoryPath, validate.EncodePathSegment(appID))).
			Desc("List release history — NOT YET ON OPENAPI GATEWAY (upstream PSM path shown; real call returns 'unavailable' until publishAPIWired=true)").
			Body(buildHistoryBody(appID, rctx.Int("limit"), strings.TrimSpace(rctx.Str("page-token"))))
		dry.Set("gateway_status", "not_deployed")
		dry.Set("note", "endpoint not yet on OpenAPI gateway; url is the upstream PSM reference, not a gateway path")
		return dry
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if err := ensurePublishWired(); err != nil {
			return err
		}
		appID := strings.TrimSpace(rctx.Str("app-id"))
		path := fmt.Sprintf(publishHistoryPath, validate.EncodePathSegment(appID))
		body := buildHistoryBody(appID, rctx.Int("limit"), strings.TrimSpace(rctx.Str("page-token")))
		data, err := rctx.CallAPI("POST", path, nil, body)
		if err != nil {
			return err
		}
		instances, _ := data["instances"].([]interface{})
		for _, it := range instances {
			if m, ok := it.(map[string]interface{}); ok {
				injectStatusName(m)
			}
		}
		rctx.OutFormat(data, nil, func(w io.Writer) {
			rows := make([]map[string]interface{}, 0, len(instances))
			for _, it := range instances {
				m, ok := it.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, map[string]interface{}{
					"ID":          m["ID"],
					"status_name": m["status_name"],
					"creator":     m["creator"],
					"createdAt":   m["createdAt"],
				})
			}
			output.PrintTable(w, rows)
		})
		return nil
	},
}

// buildHistoryBody builds the list-instances body. limit is sent only when > 0
// (0 means "not provided"); pageToken only when non-empty.
func buildHistoryBody(appID string, limit int, pageToken string) map[string]interface{} {
	body := map[string]interface{}{"appID": appID}
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

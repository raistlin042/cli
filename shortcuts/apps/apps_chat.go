// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
)

// AppsChat sends a user message to a session, starting/continuing a conversation.
// Returns turn_id (async handle); poll +session-read for status.
var AppsChat = common.Shortcut{
	Service:     appsService,
	Command:     "+chat",
	Description: "Send a message to a session to start/continue a conversation",
	Risk:        "write",
	Scopes:      []string{"spark:app:write"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "app ID", Required: true},
		{Name: "session-id", Desc: "session ID", Required: true},
		{Name: "message", Desc: "user message text", Required: true},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return output.ErrValidation("--app-id is required")
		}
		if strings.TrimSpace(rctx.Str("session-id")) == "" {
			return output.ErrValidation("--session-id is required")
		}
		// Do not echo --message content in the error (spec §4 redaction).
		if strings.TrimSpace(rctx.Str("message")) == "" {
			return output.ErrValidation("--message is required")
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		return common.NewDryRunAPI().
			POST(messagesPath(rctx.Str("app-id"), rctx.Str("session-id"))).
			Desc("Send a message to a session").
			Body(buildChatBody(rctx))
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		data, err := rctx.CallAPI("POST", messagesPath(rctx.Str("app-id"), rctx.Str("session-id")), nil, buildChatBody(rctx))
		if err != nil {
			return err
		}
		rctx.OutFormat(data, nil, func(w io.Writer) {
			fmt.Fprintf(w, "turn started: %s (poll after %vms)\n",
				common.GetString(data, "turn_id"), data["next_poll_after_ms"])
		})
		return nil
	},
}

func messagesPath(appID, sessionID string) string {
	return sessionPath(appID, sessionID) + "/messages"
}

func buildChatBody(rctx *common.RuntimeContext) map[string]interface{} {
	return map[string]interface{}{
		"message": strings.TrimSpace(rctx.Str("message")),
	}
}

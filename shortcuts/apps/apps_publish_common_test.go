// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestEnsurePublishWired_NotDeployed(t *testing.T) {
	if publishAPIWired {
		t.Skip("publishAPIWired is true; guard test only meaningful while not deployed")
	}
	err := ensurePublishWired()
	if err == nil {
		t.Fatal("ensurePublishWired() = nil, want unavailable error")
	}
}

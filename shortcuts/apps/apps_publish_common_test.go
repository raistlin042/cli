// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"
	"testing"
)

func TestNodeStatusName(t *testing.T) {
	cases := map[int]string{
		0: "Unspecified", 1: "ToDo", 2: "Running", 3: "Success",
		4: "Failed", 5: "Canceled", 6: "HoldOn", 99: "Unknown(99)",
	}
	for in, want := range cases {
		if got := nodeStatusName(in); got != want {
			t.Errorf("nodeStatusName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestEnsurePublishWired_NotDeployed(t *testing.T) {
	if publishAPIWired {
		t.Skip("publishAPIWired is true; guard test only meaningful while not deployed")
	}
	err := ensurePublishWired()
	if err == nil {
		t.Fatal("ensurePublishWired() = nil, want unavailable error")
	}
}

func TestInjectStatusName(t *testing.T) {
	m := map[string]interface{}{"status": float64(4)}
	injectStatusName(m)
	if m["status_name"] != "Failed" {
		t.Errorf("status_name = %v, want Failed", m["status_name"])
	}
	injectStatusName(nil)
	m2 := map[string]interface{}{"x": 1}
	injectStatusName(m2)
	if _, ok := m2["status_name"]; ok {
		t.Error("status_name should not be set when status is absent")
	}
}

func TestInjectStatusName_JSONNumber(t *testing.T) {
	m := map[string]interface{}{"status": json.Number("3")}
	injectStatusName(m)
	if m["status_name"] != "Success" {
		t.Errorf("json.Number path: status_name = %v, want Success", m["status_name"])
	}
}

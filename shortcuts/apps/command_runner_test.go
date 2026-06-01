// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import "testing"

func TestRedactURLCredentials(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"http with userinfo", "http://x-token:PAT_abc@git.host/app_x.git", "http://***@git.host/app_x.git"},
		{"https with userinfo", "https://u:p@h/r.git", "https://***@h/r.git"},
		{"no userinfo unchanged", "http://git.host/app_x.git", "http://git.host/app_x.git"},
		{"embedded in stderr text", "fatal: unable to access 'http://u:t@h/r.git/': 401", "fatal: unable to access 'http://***@h/r.git/': 401"},
		{"empty", "", ""},
		{"non-url unchanged", "some error message", "some error message"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := redactURLCredentials(c.in); got != c.want {
				t.Errorf("redactURLCredentials(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

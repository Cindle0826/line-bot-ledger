package main

import (
	"strings"
	"testing"
)

func TestRenderPage(t *testing.T) {
	for _, name := range []string{"static/index.html.tmpl", "static/history.html.tmpl", "static/settings.html.tmpl"} {
		page, err := renderPage(name, "1234567890-abcdefgh")
		if err != nil {
			t.Fatalf("renderPage(%q): %v", name, err)
		}
		if !strings.Contains(string(page), `"1234567890-abcdefgh"`) {
			t.Errorf("%s: rendered page doesn't contain the quoted LIFF ID: %s", name, page)
		}
	}
}

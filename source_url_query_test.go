package sourcemap_test

import (
	"net/url"
	"testing"

	"github.com/go-sourcemap/sourcemap"
)

func TestSourceURLDoesNotInheritMapQuery(t *testing.T) {
	for _, mapURL := range []string{
		"https://example.com/dist/app.map?version=1",
		"https://example.com/dist/app.map#section",
		"https://example.com/dist/app.map?version=1#section",
		"https://example.com/dist/app.map?",
		"https://example.com/dist/app.map",
	} {
		t.Run(mapURL, func(t *testing.T) {
			c, err := sourcemap.Parse(mapURL, []byte(`{"version":3,"sources":["app.js"],"sourcesContent":["source text"],"names":[],"mappings":"AAAA"}`))
			if err != nil {
				t.Fatal(err)
			}
			base, _ := url.Parse(mapURL)
			relative, _ := url.Parse("app.js")
			want := base.ResolveReference(relative).String()
			source, _, _, _, ok := c.Source(1, 0)
			if !ok || source != want {
				t.Errorf("Source=%q, %v; want %q", source, ok, want)
			}
			if got := c.SourceContent(want); got != "source text" {
				t.Errorf("SourceContent=%q, want source text", got)
			}
			if c.SourcemapURL() != mapURL {
				t.Error("original map URL changed")
			}
		})
	}
}

package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSlugifyTruncatesUnicodeSafely(t *testing.T) {
	slug := Slugify(strings.Repeat("实现", 60))
	if !utf8.ValidString(slug) {
		t.Fatalf("slug is not valid utf8: %q", slug)
	}
	if strings.ContainsRune(slug, '\uFFFD') {
		t.Fatalf("slug contains replacement character: %q", slug)
	}
	if len([]rune(slug)) > 80 {
		t.Fatalf("slug rune length = %d, want <= 80", len([]rune(slug)))
	}
}

func TestTrimSlugTruncatesUnicodeSafely(t *testing.T) {
	slug := TrimSlug("phase-29-实现-github-gitee-release-provider", 14)
	if !utf8.ValidString(slug) {
		t.Fatalf("slug is not valid utf8: %q", slug)
	}
	if strings.ContainsRune(slug, '\uFFFD') {
		t.Fatalf("slug contains replacement character: %q", slug)
	}
	if len([]rune(slug)) > 14 {
		t.Fatalf("slug rune length = %d, want <= 14", len([]rune(slug)))
	}
}

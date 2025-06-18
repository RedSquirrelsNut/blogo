package utils

import (
	"fmt"
	"html"
	"regexp"
	"time"
)

// Truncate shortens a string to max runes and adds an ellipsis if needed.
//
// If the string is shorter than or equal to max, the original string is returned.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// rssDateLayouts is a list of layouts commonly used by RSS feeds for dates.
var rssDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC850,
	time.RFC3339,
	"2006-01-02 15:04:05 -0700",
	time.ANSIC,
}

// ParsePubDate parses an RSS date string using a set of known layouts.
//
// Returns the parsed time or an error if no layout matches.
func ParsePubDate(raw string) (time.Time, error) {
	var lastErr error
	for _, layout := range rssDateLayouts {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, fmt.Errorf("unrecognized date format %q: %v", raw, lastErr)
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// StripHTML removes HTML tags and unescapes HTML entities from a string.
//
// Returns the cleaned string.
func StripHTML(raw string) string {
	withoutTags := tagRe.ReplaceAllString(raw, "")
	return html.UnescapeString(withoutTags)
}

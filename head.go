package dark

import (
	"regexp"
	"strings"
)

var darkHeadRe = regexp.MustCompile(`(?s)<dark-head>(.*?)</dark-head>`)
var titleRe = regexp.MustCompile(`(?s)<title>.*?</title>`)

// extractDarkHead finds all <dark-head>...</dark-head> elements in the HTML,
// removes them, and returns the cleaned HTML along with the collected inner content.
func extractDarkHead(html string) (cleaned string, headContent string) {
	matches := darkHeadRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return html, ""
	}

	var content strings.Builder
	for _, m := range matches {
		content.WriteString(strings.TrimSpace(m[1]))
		content.WriteByte('\n')
	}

	cleaned = darkHeadRe.ReplaceAllString(html, "")
	return cleaned, strings.TrimSpace(content.String())
}

// injectIntoHead inserts content into the <head> section, just before </head>.
// If the head content contains a <title>, the existing <title> in <head> is replaced.
func injectIntoHead(html, content string) string {
	// If the injected content has a <title>, remove any existing <title> from the head.
	if titleRe.MatchString(content) {
		headEnd := strings.Index(html, "</head>")
		if headEnd >= 0 {
			headSection := html[:headEnd]
			afterHead := html[headEnd:]
			headSection = titleRe.ReplaceAllString(headSection, "")
			html = headSection + afterHead
		}
	}

	return insertBeforeTag(html, "</head>", content)
}

// stripDarkHead removes all <dark-head> elements from HTML (for htmx partials).
func stripDarkHead(html string) string {
	return darkHeadRe.ReplaceAllString(html, "")
}

// insertBeforeTag inserts content just before the first occurrence of tag (e.g. "</head>").
// If the tag is not found, content is prepended for </head> or appended for </body>.
func insertBeforeTag(html, tag, content string) string {
	if idx := strings.Index(html, tag); idx >= 0 {
		return html[:idx] + content + "\n" + html[idx:]
	}
	if tag == "</body>" {
		return html + content
	}
	return content + "\n" + html
}

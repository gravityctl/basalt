package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// MarkdownParser handles conversion of markdown files to HTML
type MarkdownParser struct {
	markdown goldmark.Markdown
}

// NewMarkdownParser initializes the goldmark engine with GFM + task lists + typographer
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.TaskList, extension.Typographer),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		),
	}
}

// ProcessFile reads a markdown file, extracts metadata and wiki-links,
// then returns the title, rendered HTML body, link targets, and computed rel hrefs.
// sourceRelPath is the vault-relative path of this file (e.g. "recipes/index.md").
func (p *MarkdownParser) ProcessFile(filePath, sourceRelPath string) (title string, htmlBody []byte, linkTargets []string, linkHrefs []string, err error) {
	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, nil, nil, err
	}

	title = extractTitle(rawContent)
	contentWithLinks, targets, rels := extractWikiLinks(rawContent, sourceRelPath)
	linkTargets = targets
	linkHrefs = rels

	// Strip frontmatter for rendering
	contentToRender := removeFrontmatter(contentWithLinks)

	// Convert Markdown to HTML
	var buf bytes.Buffer
	if err := p.markdown.Convert(contentToRender, &buf); err != nil {
		return "", nil, nil, nil, err
	}

	htmlBody = buf.Bytes()

	// Rewrite .md links to .html
	reLink := regexp.MustCompile(`\(([^)]*?\.md)\)`)
	htmlBody = reLink.ReplaceAll(htmlBody, []byte("($1html)"))

	return title, htmlBody, linkTargets, linkHrefs, nil
}

// extractTitle gets the title from frontmatter, first H1, or falls back to filename
func extractTitle(data []byte) string {
	// 1. Frontmatter title
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	if matches := re.FindSubmatch(data); len(matches) > 0 {
		yamlContent := string(matches[1])
		for _, line := range strings.Split(yamlContent, "\n") {
			if strings.HasPrefix(line, "title:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// 2. First H1
	h1Re := regexp.MustCompile(`(?m)^#\s+(.*)$`)
	if matches := h1Re.FindSubmatch(data); len(matches) == 2 {
		return strings.TrimSpace(string(matches[1]))
	}

	return "Untitled"
}

// toHTMLName converts a markdown file path to an HTML-safe name (strips .md extension)
func toHTMLName(mdPath string) string {
	return strings.TrimSuffix(filepath.Base(mdPath), ".md")
}

// removeFrontmatter strips YAML frontmatter from raw content
func removeFrontmatter(data []byte) []byte {
	re := regexp.MustCompile(`(?s)^---\s*\n.*?\n---\n?`)
	return re.ReplaceAll(data, []byte{})
}

// extractTags gets the tags list from frontmatter.
// Handles both array syntax: tags: [tag1, tag2]
// And multi-line syntax: tags:\n  - tag1\n  - tag2
func extractTags(data []byte) []string {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	if matches := re.FindSubmatch(data); len(matches) > 0 {
		yamlContent := string(matches[1])

		// Handle inline array: tags: [tag1, tag2, tag3]
		inlineRe := regexp.MustCompile(`(?m)^tags:\s*\[([^\]]*)\]`)
		if m := inlineRe.FindSubmatch([]byte(yamlContent)); len(m) > 0 {
			parts := strings.Split(string(m[1]), ",")
			var tags []string
			for _, p := range parts {
				t := strings.TrimSpace(p)
				if t != "" {
					tags = append(tags, t)
				}
			}
			return tags
		}

		// Handle multi-line list: tags:\n  - tag1\n  - tag2
		multilineRe := regexp.MustCompile(`(?m)^tags:\s*\n((?:\s+-\s*[^\n]+\n?)+)`)
		if m := multilineRe.FindSubmatch([]byte(yamlContent)); len(m) > 0 {
			lines := strings.Split(strings.TrimSpace(string(m[1])), "\n")
			var tags []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "-") {
					tag := strings.TrimSpace(strings.TrimPrefix(line, "-"))
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}
			return tags
		}
	}
	return nil
}

// extractDate gets the date from frontmatter if present.
func extractDate(data []byte) string {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	if matches := re.FindSubmatch(data); len(matches) > 0 {
		yamlContent := string(matches[1])
		dateRe := regexp.MustCompile(`(?m)^date:\s*["']?([^"'\n]+)["']?`)
		if m := dateRe.FindSubmatch([]byte(yamlContent)); len(m) > 0 {
			return strings.TrimSpace(string(m[1]))
		}
	}
	return ""
}

// computeReadingTime estimates reading time from word count.
// Assumes ~200 words per minute.
func computeReadingTime(htmlBody []byte) string {
	words := 0
	inTag := false
	for _, b := range htmlBody {
		switch b {
		case '<':
			inTag = true
		case '>':
			inTag = false
		case ' ', '\n', '\r', '\t':
			if !inTag {
				words++
			}
		}
	}
	minutes := (words + 199) / 200 // ceiling division
	if minutes < 1 {
		minutes = 1
	}
	if minutes == 1 {
		return "1 min read"
	}
	return fmt.Sprintf("%d min read", minutes)
}

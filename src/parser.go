package main

import (
	"bytes"
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

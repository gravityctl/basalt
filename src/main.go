package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	//"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/parser"
)

// Configuration
const (
	SourceDir = "../vault"   // Your Obsidian folder
	OutputDir = "../output"  // Where the static site goes
)

// MarkdownParser handles the conversion logic
type MarkdownParser struct {
	markdown goldmark.Markdown
}

// NewMarkdownParser initializes the goldmark engine
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

// ProcessFile reads a markdown file, extracts metadata, and returns HTML
func (p *MarkdownParser) ProcessFile(filePath string) (string, string, error) {
	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	title := extractTitle(rawContent)
	contentToRender := removeFrontmatter(rawContent)

	// Convert Markdown to HTML
	var buf bytes.Buffer
	if err := p.markdown.Convert(contentToRender, &buf); err != nil {
		return "", "", err
	}

	htmlContent := buf.String()

	// CRITICAL: Rewrite internal links from .md to .html
	// This regex finds links like [text](filename.md) and replaces .md with .html
	reLink := regexp.MustCompile(`\(([^)]*?\.md)\)`)
	htmlContent = reLink.ReplaceAllString(htmlContent, "($1html)")

	return title, htmlContent, nil
}

// Helper: Extract title from Frontmatter or file
func extractTitle(data []byte) string {
	// 1. Check for YAML Frontmatter
	re := regexp.MustCompile(`^---\s(.*\s)*---`)
	matches := re.FindSubmatch(data)
	if len(matches) > 1 {
		yamlContent := string(matches[1])
		lines := strings.Split(yamlContent, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "title:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
          fmt.Printf("Thing '%s'.\n", strings.TrimSpace(parts[1]))
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// 2. Fallback: First H1 in document
	h1Re := regexp.MustCompile(`^#\s+(.*)$`)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := h1Re.FindStringSubmatch(line); len(matches) == 2 {
			return strings.TrimSpace(matches[1])
		}
	}

	// 3. Fallback: Filename
	return strings.TrimSuffix(filepath.Base("error"), ".md")
}

// Helper: Remove frontmatter for rendering
func removeFrontmatter(data []byte) []byte {
	re := regexp.MustCompile(`(?s)^---\s*\n.*?\n---\n?`)
	return re.ReplaceAll(data, []byte{})
}

// Helper: Clean filename for HTML
func toHTMLName(mdPath string) string {
	base := strings.TrimSuffix(filepath.Base(mdPath), ".md")
	return base + ".html"
}

func main() {
	parser := NewMarkdownParser()

	// Check if source exists
	if _, err := os.Stat(SourceDir); os.IsNotExist(err) {
		fmt.Printf("Error: Source directory '%s' not found.\n", SourceDir)
		fmt.Println("Please create this folder and add your Obsidian .md files.")
		return
	}

	// Create output directory
	err := os.MkdirAll(OutputDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}

	fmt.Println("Building Basalt Site...")

	// Walk through the directory
	err = filepath.Walk(SourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .md files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Process content
		title, htmlBody, err := parser.ProcessFile(path)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", path, err)
			return nil // skip file and continue
		}

		// Generate Output Filename
		relPath, _ := filepath.Rel(SourceDir, path)
		outputDir := filepath.Join(OutputDir, filepath.Dir(relPath))
		os.MkdirAll(outputDir, 0755)

		outputFile := filepath.Join(outputDir, toHTMLName(relPath))

		// Generate the HTML Template
		finalHTML := generateHTMLTemplate(title, htmlBody, relPath)

		// Write to disk
		err = os.WriteFile(outputFile, []byte(finalHTML), 0644)
		if err != nil {
			return err
		}

		fmt.Printf("Generated: %s -> %s\n", relPath, outputFile)

		return nil
	})

	if err != nil {
		fmt.Printf("Walk error: %v\n", err)
	}

	fmt.Println("Build complete.")
}

// generateHTMLTemplate creates a simple, clean HTML wrapper
func generateHTMLTemplate(title string, content string, sourcePath string) string {
	css := `
	:root { --bg: #f8f8f8; --text: #333; --link: #2980b9; }
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; max-width: 850px; margin: 0 auto; padding: 20px; background: var(--bg); color: var(--text); }
	h1 { border-bottom: 1px solid #e1e4e8; padding-bottom: 10px; }
	a { color: var(--link); text-decoration: none; font-weight: 500; }
	a:hover { text-decoration: underline; }
	nav { margin-bottom: 20px; font-size: 0.85em; color: #666; }
	.content { margin-top: 20px; }
	.markdown-body { background: white; padding: 30px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
	p, li { font-size: 16px; }
	h2 { margin-top: 30px; }
	`

	navHTML := ""
	if sourcePath != "" {
		navHTML = fmt.Sprintf("<nav>Source: <code>%s</code></nav>", sourcePath)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Basalt</title>
    <style>%s</style>
</head>
<body>
    <header>
        <h1>%s</h1>
        %s
    </header>
    <main class="content">
        <div class="markdown-body">
            %s
        </div>
    </main>
</body>
</html>`, title, css, title, navHTML, content)
}

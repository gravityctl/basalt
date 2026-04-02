package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config — adjust these to match your vault layout
const (
	SourceDir = "../vault"
	OutputDir = "../output"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat(SourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory %q not found", SourceDir)
	}

	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	graphDir := filepath.Join(OutputDir, "graph")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		return fmt.Errorf("creating graph dir: %w", err)
	}

	fmt.Println("Building Basalt Site...")

	// Build full vault graph (computes all pages, edges, writes backlinks.json)
	graph, _, pageTitles, err := buildGraph(SourceDir)
	if err != nil {
		return fmt.Errorf("building graph: %w", err)
	}

	graphJSON, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling graph: %w", err)
	}
	if err := os.WriteFile(filepath.Join(OutputDir, "graph.json"), graphJSON, 0644); err != nil {
		return fmt.Errorf("writing graph.json: %w", err)
	}
	fmt.Printf("Graph: %d nodes, %d edges\n", len(graph.Nodes), len(graph.Edges))

	// Load the set of all existing pages for stub detection
	existingPages := make(map[string]bool)
	for _, node := range graph.Nodes {
		if !node.Stub {
			existingPages[node.ID] = true
		}
	}

	// Load backlinks map for per-page backlink lookup
	backlinksMap := loadBacklinks()

	// Build navigation tree for left sidebar
	navTree := buildNavTree(SourceDir)
	navTreeJSON, _ := json.Marshal(navTree)

	parser := NewMarkdownParser()

	// Walk the vault and generate HTML for each markdown file
	err = filepath.Walk(SourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(SourceDir, path)

		// Read raw content for tag extraction before ProcessFile strips frontmatter
		rawContent, _ := os.ReadFile(path)
		tags := extractTags(rawContent)
		date := extractDate(rawContent)

		title, htmlBody, linkTargets, linkHrefs, err := parser.ProcessFile(path, relPath)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", path, err)
			return nil
		}

		readingTime := computeReadingTime(htmlBody)

		pageID := filepath.Join(filepath.Dir(relPath), toHTMLName(relPath))
		outputSubdir := filepath.Join(OutputDir, filepath.Dir(relPath))
		if merr := os.MkdirAll(outputSubdir, 0755); merr != nil {
			return merr
		}

		// Build per-page graph data
		pageGraph := buildPageGraph(pageID, linkTargets, linkHrefs, backlinksMap, existingPages, pageTitles, tags)
		pageGraph.CurrentHref = pageID + ".html"
		pageGraph.TableOfContents = extractTOC(htmlBody)
		pageGraph.Date = date
		pageGraph.ReadingTime = readingTime

		// Write HTML page
		outputFile := filepath.Join(OutputDir, pageID+".html")
		html := generateHTMLTemplate(title, string(htmlBody), relPath, pageGraph, string(navTreeJSON))
		if werr := os.WriteFile(outputFile, []byte(html), 0644); werr != nil {
			return werr
		}
		fmt.Printf("Generated: %s\n", outputFile)

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking vault: %w", err)
	}

	// Generate stub pages for dead link targets
	stubCount := 0
	for _, node := range graph.Nodes {
		if node.Stub {
			stubFile := filepath.Join(OutputDir, node.ID+".html")
			if err := os.WriteFile(stubFile, []byte(generateStubHTML(node.ID)), 0644); err != nil {
				return err
			}
			fmt.Printf("Stubbed: %s\n", stubFile)
			stubCount++
		}
	}
	if stubCount > 0 {
		fmt.Printf("Generated %d stub pages\n", stubCount)
	}

	writeGraphViewer(graphDir, graphJSON)
	fmt.Println("Build complete.")
	return nil
}

// buildPageGraph builds the per-page graph data for a given page:
// - Links: pages this page wiki-links to
// - Backlinks: pages that link to this page
func buildPageGraph(pageID string, linkTargets []string, linkHrefs []string, backlinksMap map[string][]string, existingPages map[string]bool, pageTitles map[string]string, tags []string) *PageGraph {
	pg := &PageGraph{Links: []GraphRef{}, Backlinks: []GraphRef{}, Tags: tags}

	// Build Links — use linkHrefs (computed relative hrefs) not bare target paths
	for i, target := range linkTargets {
		href := target + ".html" // fallback
		if i < len(linkHrefs) {
			href = linkHrefs[i] + ".html"
		}
		pg.Links = append(pg.Links, GraphRef{
			Title: toHTMLName(target),
			Href:  href,
			Stub:  !existingPages[target],
		})
	}

	// Build Backlinks — compute relative hrefs from this page's directory
	for _, source := range backlinksMap[pageID] {
		title := toHTMLName(source)
		if t, ok := pageTitles[source]; ok && t != "" {
			title = t
		}
		pg.Backlinks = append(pg.Backlinks, GraphRef{
			Title: title,
			Href:  computeRelHref(pageID, source),
			Stub:  !existingPages[source],
		})
	}

	return pg
}

// loadBacklinks reads the backlinks map from backlinks.json
func loadBacklinks() map[string][]string {
	var m map[string][]string
	data, err := os.ReadFile(filepath.Join(OutputDir, "backlinks.json"))
	if err != nil {
		return make(map[string][]string)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string][]string)
	}
	return m
}

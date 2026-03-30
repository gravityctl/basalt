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
	graph, pageLinks, err := buildGraph(SourceDir)
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

	parser := NewMarkdownParser()

	// Walk the vault and generate HTML for each markdown file
	err = filepath.Walk(SourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		title, htmlBody, linkTargets, err := parser.ProcessFile(path)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", path, err)
			return nil
		}

		relPath, _ := filepath.Rel(SourceDir, path)
		pageID := filepath.Join(filepath.Dir(relPath), toHTMLName(relPath))

		outputSubdir := filepath.Join(OutputDir, filepath.Dir(relPath))
		if err := os.MkdirAll(outputSubdir, 0755); err != nil {
			return err
		}

		// Build per-page graph data
		pageGraph := buildPageGraph(pageID, linkTargets, backlinksMap, existingPages)

		// Write HTML page
		outputFile := filepath.Join(OutputDir, pageID+".html")
		html := generateHTMLTemplate(title, string(htmlBody), relPath, pageGraph)
		if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
			return err
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

	writeGraphViewer(graphDir, len(graph.Nodes))
	fmt.Println("Build complete.")
	return nil
}

// buildPageGraph builds the per-page graph data for a given page:
// - Links: pages this page wiki-links to
// - Backlinks: pages that link to this page
func buildPageGraph(pageID string, linkTargets []string, backlinksMap map[string][]string, existingPages map[string]bool) *PageGraph {
	pg := &PageGraph{Links: []GraphRef{}, Backlinks: []GraphRef{}}

	// Build Links
	for _, target := range linkTargets {
		pg.Links = append(pg.Links, GraphRef{
			Title: toHTMLName(target),
			Href:  target + ".html",
			Stub:  !existingPages[target],
		})
	}

	// Build Backlinks
	for _, source := range backlinksMap[pageID] {
		pg.Backlinks = append(pg.Backlinks, GraphRef{
			Title: toHTMLName(source),
			Href:  source + ".html",
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

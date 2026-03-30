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
	SourceDir = "../vault"   // Your Obsidian folder
	OutputDir = "../output"  // Where the static site goes
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

	// Build full vault graph
	graph, pageLinks, err := buildGraph(SourceDir)
	if err != nil {
		return fmt.Errorf("building graph: %w", err)
	}

	if err := writeGraphJSON(graph); err != nil {
		return fmt.Errorf("writing graph.json: %w", err)
	}
	fmt.Printf("Graph: %d nodes, %d edges\n", len(graph.Nodes), len(graph.Edges))

	// Track pages we generate so we know which graph targets are stubs
	generatedPages := make(map[string]bool)

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
			return nil // skip and continue
		}

		relPath, _ := filepath.Rel(SourceDir, path)
		// Preserve subdirectory: recipes/index.md -> recipes/index
		pageID := filepath.Join(filepath.Dir(relPath), toHTMLName(relPath))

		// Ensure output directory exists for this page
		outputSubdir := filepath.Join(OutputDir, filepath.Dir(relPath))
		if err := os.MkdirAll(outputSubdir, 0755); err != nil {
			return err
		}

		// Per-page graph data
		pageGraph := buildPageGraph(pageID, linkTargets, pageLinks, generatedPages)

		// Write HTML page
		outputFile := filepath.Join(OutputDir, pageID+".html")
		html := generateHTMLTemplate(title, string(htmlBody), relPath, pageGraph)
		if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
			return err
		}
		fmt.Printf("Generated: %s\n", outputFile)
		generatedPages[pageID] = true

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking vault: %w", err)
	}

	// Generate stub pages for dead link targets
	stubCount := 0
	for _, node := range graph.Nodes {
		if node.Stub && !generatedPages[node.ID] {
			stubFile := filepath.Join(OutputDir, node.ID+".html")
			if err := os.WriteFile(stubFile, []byte(generateStubHTML(node.ID)), 0644); err != nil {
				return err
			}
			fmt.Printf("Stubbed: %s (dead link target)\n", stubFile)
			stubCount++
		}
	}
	if stubCount > 0 {
		fmt.Printf("Generated %d stub pages\n", stubCount)
	}

	// Write D3 graph viewer
	writeGraphViewer(graphDir, len(graph.Nodes))
	fmt.Println("Build complete.")
	return nil
}

// writeGraphJSON writes the full vault graph as JSON
func writeGraphJSON(g *Graph) error {
	graphJSON, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(OutputDir, "graph.json"), graphJSON, 0644)
}

// buildPageGraph constructs per-page graph data (links + backlinks) for a given page
func buildPageGraph(pageID string, linkTargets []string, pageLinks map[string][]string, generatedPages map[string]bool) *PageGraph {
	pg := &PageGraph{Links: []GraphRef{}, Backlinks: []GraphRef{}}

	// Resolve link targets
	for _, target := range linkTargets {
		// target is already the full path without extension (e.g. "recipes/mac")
		// Use toHTMLName only for the display title (just the filename)
		pg.Links = append(pg.Links, GraphRef{
			Title: toHTMLName(target),
			Href:  target + ".html",
			Stub:  !generatedPages[target],
		})
	}

	// Load global backlinks map
	backlinksMap := loadBacklinks()
	for _, source := range backlinksMap[pageID] {
		pg.Backlinks = append(pg.Backlinks, GraphRef{
			Title: toHTMLName(source),
			Href:  source + ".html",
		})
	}

	return pg
}

// loadBacklinks reads the backlinks.json file into a map
func loadBacklinks() map[string][]string {
	var m map[string][]string
	data, err := os.ReadFile(filepath.Join(OutputDir, "backlinks.json"))
	if err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}

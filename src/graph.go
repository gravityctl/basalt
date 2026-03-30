package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Graph represents the full vault graph
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode is a page in the graph
type GraphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
	Stub  bool   `json:"stub"` // true if this was a dead link we stubbed
}

// GraphEdge represents a link from one page to another
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// PageGraph is the per-page graph data injected into each page
type PageGraph struct {
	Links     []GraphRef `json:"links"`
	Backlinks []GraphRef `json:"backlinks"`
}

// GraphRef is a reference to another page
type GraphRef struct {
	Title string `json:"title"`
	Href  string `json:"href"`
	Stub  bool   `json:"stub"` // true if target doesn't exist
}

// wikiLinkRe matches [[Page]] and [[Page|Display Text]]
var wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// extractWikiLinks extracts all wiki-style links from content.
// sourceRelPath is the relative path of the source file within the vault (e.g. "recipes/index.md").
// It returns the cleaned content (wiki links → markdown links) and the raw link targets.
// Href paths are computed relative to the source file's directory in the output.
func extractWikiLinks(content []byte, sourceRelPath string) ([]byte, []string) {
	links := []string{}

	processed := wikiLinkRe.ReplaceAllFunc(content, func(match []byte) []byte {
		matches := wikiLinkRe.FindSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		target := string(matches[1])
		display := target
		if len(matches) >= 3 && len(matches[2]) > 0 {
			display = string(matches[2])
		}

		links = append(links, target)

		// Compute href relative to source file's directory in output.
		// sourcePageID e.g. "recipes/index" (no extension)
		sourcePageID := strings.TrimSuffix(sourceRelPath, ".md")
		sourceDir := filepath.Dir(sourcePageID) // "." for root files, "recipes" for files in subdirs
		targetBase := toHTMLName(target)        // filename only, no dir
		targetDir := filepath.Dir(target)       // directory part of wiki-link target

		// Relative path from source's output directory to target's output file
		rel, err := filepath.Rel(sourceDir, filepath.Join(targetDir, targetBase))
		if err != nil {
			rel = targetBase
		}

		// Build display text: use filename only (toHTMLName), or explicit display text if provided
		linkDisplay := toHTMLName(target)
		if len(matches) >= 3 && len(matches[2]) > 0 {
			linkDisplay = display
		}

		return []byte("[" + linkDisplay + "](" + rel + ".html)")
	})

	return processed, links
}

// buildGraph walks the vault and builds the complete link graph.
// Returns the graph, and a backlinks map (target pageID -> list of source pageIDs).
func buildGraph(vaultDir string) (*Graph, map[string][]string, error) {
	g := &Graph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}
	allPages := make(map[string]bool)

	// First pass: collect all existing pages
	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		relPath, _ := filepath.Rel(vaultDir, path)
		pageID := pageIDFromRelPath(relPath)
		allPages[pageID] = true
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Second pass: extract links from each page
	err = filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(vaultDir, path)
		sourceID := pageIDFromRelPath(relPath)

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		_, targets := extractWikiLinks(data, relPath)

		// Add edges
		for _, target := range targets {
			g.Edges = append(g.Edges, GraphEdge{
				Source: sourceID,
				Target: target,
			})
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Build nodes from all pages + stub pages (dead link targets)
	addedNodes := make(map[string]bool)
	for pageID := range allPages {
		g.Nodes = append(g.Nodes, GraphNode{
			ID:    pageID,
			Title: toHTMLName(pageID),
			Path:  pageID + ".html",
			Stub:  false,
		})
		addedNodes[pageID] = true
	}

	// Add stub nodes for dead links
	for _, edges := range g.Edges {
		target := edges.Target
		if !addedNodes[target] {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:    target,
				Title: toHTMLName(target),
				Path:  target + ".html",
				Stub:  true,
			})
			addedNodes[target] = true
		}
	}

	// Compute backlinks: for each page, who links to it?
	backlinks := make(map[string][]string)
	for _, edge := range g.Edges {
		backlinks[edge.Target] = append(backlinks[edge.Target], edge.Source)
	}

	// Store backlinks for per-page graph queries
	backlinksJSON, _ := json.MarshalIndent(backlinks, "", "  ")
	os.WriteFile(filepath.Join(vaultDir, "..", "output", "backlinks.json"), backlinksJSON, 0644)

	return g, backlinks, nil
}

// pageIDFromRelPath converts a vault-relative path (e.g. "recipes/index.md")
// to a page ID (e.g. "recipes/index") preserving directory structure.
func pageIDFromRelPath(relPath string) string {
	// relPath is like "recipes/index.md"
	// pageID should be "recipes/index"
	dir := filepath.Dir(relPath)        // "recipes"
	base := toHTMLName(relPath)          // "index" (strips .md)
	if dir == "." {
		return base
	}
	return filepath.Join(dir, base)
}

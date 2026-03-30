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

// extractWikiLinks extracts all wiki-style links from content
// Returns the cleaned content (with wiki links converted to markdown links)
// and a list of all link targets found.
func extractWikiLinks(content []byte) ([]byte, []string) {
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
		// Convert to standard markdown link
		return []byte("[" + display + "](../" + toHTMLName(target) + ")")
	})

	return processed, links
}

// buildGraph walks the vault and builds the complete link graph
func buildGraph(vaultDir string) (*Graph, map[string][]string, error) {
	g := &Graph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}
	// pageLinks: page path -> list of pages it links to
	pageLinks := make(map[string][]string)
	// allPages: set of all page IDs (for backlink computation)
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
		pageID := toHTMLName(strings.TrimSuffix(relPath, ".html"))
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
		sourceID := toHTMLName(strings.TrimSuffix(relPath, ".html"))

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		_, links := extractWikiLinks(data)

		pageLinks[sourceID] = links

		// Add edges
		for _, target := range links {
			targetID := toHTMLName(target)
			g.Edges = append(g.Edges, GraphEdge{
				Source: sourceID,
				Target: targetID,
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
			Title: pageID,
			Path:  pageID + ".html",
			Stub:  false,
		})
		addedNodes[pageID] = true
	}

	// Add stub nodes for dead links
	for _, links := range pageLinks {
		for _, target := range links {
			targetID := toHTMLName(target)
			if !addedNodes[targetID] {
				g.Nodes = append(g.Nodes, GraphNode{
					ID:    targetID,
					Title: targetID,
					Path:  targetID + ".html",
					Stub:  true,
				})
				addedNodes[targetID] = true
			}
		}
	}

	// Compute backlinks: for each page, who links to it?
	backlinks := make(map[string][]string)
	for source, links := range pageLinks {
		for _, target := range links {
			targetID := toHTMLName(target)
			backlinks[targetID] = append(backlinks[targetID], source)
		}
	}

	// Store backlinks in a global JSON for per-page graph queries
	backlinksJSON, _ := json.Marshal(backlinks)
	os.WriteFile(filepath.Join(vaultDir, "..", "output", "backlinks.json"), backlinksJSON, 0644)

	return g, pageLinks, nil
}

// generatePageGraph builds PageGraph for a specific page
func generatePageGraph(pageID string, pageLinks map[string][]string, vaultDir string) *PageGraph {
	links := pageLinks[pageID]
	backlinksMap := make(map[string][]string)
	// Read from the global backlinks.json
	data, err := os.ReadFile(filepath.Join(vaultDir, "..", "output", "backlinks.json"))
	if err == nil {
		json.Unmarshal(data, &backlinksMap)
	}

	pg := &PageGraph{
		Links:     []GraphRef{},
		Backlinks: []GraphRef{},
	}

	// Determine which pages actually exist
	existingPages := make(map[string]bool)
	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".md") {
			relPath, _ := filepath.Rel(vaultDir, path)
			pageID := toHTMLName(strings.TrimSuffix(relPath, ".html"))
			existingPages[pageID] = true
		}
		return nil
	})

	// Build Links
	for _, target := range links {
		targetID := toHTMLName(target)
		stub := !existingPages[targetID]
		pg.Links = append(pg.Links, GraphRef{
			Title: target,
			Href:  targetID + ".html",
			Stub:  stub,
		})
	}

	// Build Backlinks
	for _, source := range backlinksMap[pageID] {
		pg.Backlinks = append(pg.Backlinks, GraphRef{
			Title: source,
			Href:  source + ".html",
			Stub:  false,
		})
	}

	return pg
}

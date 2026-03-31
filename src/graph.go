package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// NavNode is a node in the navigation tree
type NavNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"` // full pageID (e.g. "recipes/tacos"), empty for folders
	Href     string     `json:"href"` // HTML href, empty for folders
	Children []*NavNode `json:"children,omitempty"`
}

// wikiLinkRe matches [[Page]] and [[Page|Display Text]]
var wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// extractWikiLinks extracts all wiki-style links from content.
// sourceRelPath is the relative path of the source file within the vault (e.g. "recipes/index.md").
// Returns the cleaned content (wiki links → markdown links), raw link targets, and the computed
// relative hrefs for each link.
func extractWikiLinks(content []byte, sourceRelPath string) ([]byte, []string, []string) {
	targets := []string{}
	rels := []string{}

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

		targets = append(targets, target)

		// Compute href relative to source file's directory in output.
		sourcePageID := strings.TrimSuffix(sourceRelPath, ".md")
		sourceDir := filepath.Dir(sourcePageID)
		targetBase := toHTMLName(target)
		targetDir := filepath.Dir(target)

		rel, err := filepath.Rel(sourceDir, filepath.Join(targetDir, targetBase))
		if err != nil {
			rel = targetBase
		}
		rels = append(rels, rel)

		linkDisplay := toHTMLName(target)
		if len(matches) >= 3 && len(matches[2]) > 0 {
			linkDisplay = display
		}

		return []byte("[" + linkDisplay + "](" + rel + ".html)")
	})

	return processed, targets, rels
}

// computeRelHref computes the relative href from a source page to a target page.
func computeRelHref(sourcePageID, targetPageID string) string {
	sourceDir := filepath.Dir(sourcePageID)
	rel, err := filepath.Rel(sourceDir, targetPageID)
	if err != nil {
		return targetPageID + ".html"
	}
	return rel + ".html"
}

// buildNavTree builds a hierarchical navigation tree from all pages.
// Returns a sorted list of NavNode trees (folders at top level, pages as leaves).
func buildNavTree(vaultDir string) []*NavNode {
	// Collect all page paths
	var allPaths []string
	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") {
			return nil
		}
		relPath, _ := filepath.Rel(vaultDir, path)
		allPaths = append(allPaths, relPath)
		return nil
	})

	// Build tree
	root := &NavNode{Name: "", Children: []*NavNode{}}
	for _, relPath := range allPaths {
		pageID := strings.TrimSuffix(relPath, ".md") // "recipes/tacos"
		title := toHTMLName(pageID)
		href := pageID + ".html"

		// Get frontmatter title if available
		fullPath := filepath.Join(vaultDir, relPath)
		if data, err := os.ReadFile(fullPath); err == nil {
			if t := extractTitle(data); t != "" && t != "Untitled" {
				title = t
			}
		}

		parts := strings.Split(pageID, "/")
		node := &NavNode{Name: title, Path: pageID, Href: href}

		// Navigate/create the tree
		current := root
		for i := 0; i < len(parts)-1; i++ {
			dir := parts[i]
			// Find or create folder node
			found := false
			for _, child := range current.Children {
				if child.Name == dir && child.Path == "" {
					current = child
					found = true
					break
				}
			}
			if !found {
				folder := &NavNode{Name: dir, Children: []*NavNode{}}
				current.Children = append(current.Children, folder)
				current = folder
			}
		}

		// Add the page as a leaf
		current.Children = append(current.Children, node)
	}
	// Sort: home page first, then folders (alphabetical), then other pages (alphabetical)
	var sortNodes func([]*NavNode)
	sortNodes = func(nodes []*NavNode) {
		sort.Slice(nodes, func(i, j int) bool {
			a, b := nodes[i], nodes[j]
			aIsHome := a.Path == "index"
			bIsHome := b.Path == "index"
			if aIsHome != bIsHome {
				return aIsHome
			}
			aIsFolder := a.Href == ""
			bIsFolder := b.Href == ""
			if aIsFolder != bIsFolder {
				return aIsFolder
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		})
		for _, n := range nodes {
			if n.Children != nil {
				sortNodes(n.Children)
			}
		}
	}
	}
	sortNodes(root.Children)

	return root.Children
}

// buildGraph walks the vault and builds the complete link graph.
func buildGraph(vaultDir string) (*Graph, map[string][]string, map[string]string, error) {
	g := &Graph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	allPages := make(map[string]bool)
	pageTitles := make(map[string]string)

	// First pass: collect pages and titles
	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") {
			return nil
		}
		relPath, _ := filepath.Rel(vaultDir, path)
		pageID := strings.TrimSuffix(relPath, ".md")
		allPages[pageID] = true
		if data, err := os.ReadFile(path); err == nil {
			pageTitles[pageID] = extractTitle(data)
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	// Second pass: extract links
	err = filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") {
			return nil
		}
		relPath, _ := filepath.Rel(vaultDir, path)
		sourceID := strings.TrimSuffix(relPath, ".md")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		_, targets, _ := extractWikiLinks(data, relPath)
		for _, target := range targets {
			g.Edges = append(g.Edges, GraphEdge{Source: sourceID, Target: target})
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	addedNodes := make(map[string]bool)
	for pageID := range allPages {
		title := pageTitles[pageID]
		if title == "" {
			title = toHTMLName(pageID)
		}
		g.Nodes = append(g.Nodes, GraphNode{
			ID: pageID, Title: title, Path: pageID + ".html", Stub: false,
		})
		addedNodes[pageID] = true
	}

	for _, edge := range g.Edges {
		if !addedNodes[edge.Target] {
			g.Nodes = append(g.Nodes, GraphNode{
				ID: edge.Target, Title: toHTMLName(edge.Target),
				Path: edge.Target + ".html", Stub: true,
			})
			addedNodes[edge.Target] = true
		}
	}

	backlinks := make(map[string][]string)
	for _, edge := range g.Edges {
		backlinks[edge.Target] = append(backlinks[edge.Target], edge.Source)
	}

	backlinksJSON, _ := json.Marshal(backlinks)
	os.WriteFile(filepath.Join(vaultDir, "..", "output", "backlinks.json"), backlinksJSON, 0644)

	return g, backlinks, pageTitles, nil
}

// pageIDFromRelPath converts a vault-relative path (e.g. "recipes/index.md")
// to a page ID (e.g. "recipes/index") preserving directory structure.
func pageIDFromRelPath(relPath string) string {
	dir := filepath.Dir(relPath)
	base := toHTMLName(relPath)
	if dir == "." {
		return base
	}
	return filepath.Join(dir, base)
}

// downloadD3 fetches D3 from CDN and saves it locally to graphDir/d3.min.js
func downloadD3(graphDir string) {
	d3Path := filepath.Join(graphDir, "d3.min.js")
	if _, err := os.Stat(d3Path); err == nil {
		return
	}
	resp, err := http.Get("https://cdn.jsdelivr.net/npm/d3@7/dist/d3.min.js")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	os.WriteFile(d3Path, data, 0644)
}

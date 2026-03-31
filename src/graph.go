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
	Stub  bool   `json:"stub"`
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
	Stub  bool   `json:"stub"`
}

// NavNode is a node in the navigation tree
type NavNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Href     string     `json:"href"`
	Children []*NavNode `json:"children,omitempty"`
}

// wikiLinkRe matches [[Page]] and [[Page|Display Text]]
var wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+)?\]\]`)

// extractWikiLinks extracts wiki-style links from markdown content.
func extractWikiLinks(content []byte, sourceRelPath string) ([]byte, []string, []string, error) {
	var targets, rels []string
	result := wikiLinkRe.ReplaceAllFunc(content, func(match []byte) []byte {
		m := wikiLinkRe.FindSubmatch(match)
		if len(m) < 2 {
			return match
		}
		target := string(m[1])
		display := target
		if len(m) >= 3 && len(m[3]) > 0 {
			display = string(m[3])
		}
		targets = append(targets, target)

		srcDir := filepath.Dir(strings.TrimSuffix(sourceRelPath, ".md"))
		tgtBase := toHTMLName(target)
		tgtDir := filepath.Dir(target)
		rel, err := filepath.Rel(srcDir, filepath.Join(tgtDir, tgtBase))
		if err != nil {
			rel = tgtBase
		}
		rels = append(rels, rel)

		linkDisp := toHTMLName(target)
		if len(m) >= 3 && len(m[3]) > 0 {
			linkDisp = display
		}
		return []byte("[" + linkDisp + "](" + rel + ".html)")
	})
	return result, targets, rels, nil
}

// computeRelHref computes relative href from source page to target page.
func computeRelHref(sourcePageID, targetPageID string) string {
	rel, err := filepath.Dir(sourcePageID), targetPageID
	r, err := filepath.Rel(rel, targetPageID)
	if err != nil {
		return targetPageID + ".html"
	}
	return r + ".html"
}

// buildNavTree walks the vault and builds a hierarchical nav tree.
// "index" page is always first; folders sorted alphabetically; pages sorted alphabetically.
func buildNavTree(vaultDir string) []*NavNode {
	type entry struct{ pageID, title string }
	entries := []entry{}

	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(vaultDir, path)
		pageID := strings.TrimSuffix(rel, ".md")
		title := toHTMLName(pageID)
		if data, readErr := os.ReadFile(path); readErr == nil {
			if t := extractTitle(data); t != "" && t != "Untitled" {
				title = t
			}
		}
		entries = append(entries, entry{pageID: pageID, title: title})
		return nil
	})

	type tn struct {
		name     string
		path     string
		href     string
		children map[string]*tn
	}

	root := map[string]*tn{"": {name: "", path: "", href: "", children: map[string]*tn{}}}

	for _, e := range entries {
		parts := strings.Split(e.pageID, "/")
		cur := root[""]
		for i, part := range parts {
			if cur.children[part] == nil {
				isLeaf := i == len(parts)-1
				child := &tn{
					name:     e.title,
					path:     e.pageID,
					href:     e.pageID + ".html",
					children: map[string]*tn{},
				}
				if !isLeaf {
					child.name = part
					child.path = ""
					child.href = ""
				}
				cur.children[part] = child
			}
			cur = cur.children[part]
		}
	}

	var flatten func(m map[string]*tn) []*NavNode
	flatten = func(m map[string]*tn) []*NavNode {
		var dirs, pages []*NavNode
		for _, c := range m {
			if c.href == "" {
				dirs = append(dirs, &NavNode{Name: c.name, Children: flatten(c.children)})
			} else {
				pages = append(pages, &NavNode{Name: c.name, Path: c.path, Href: c.href})
			}
		}
		sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
		sort.Slice(pages, func(i, j int) bool {
			if pages[i].Name == "index" {
				return true
			}
			if pages[j].Name == "index" {
				return false
			}
			return pages[i].Name < pages[j].Name
		})
		return append(dirs, pages...)
	}

	return flatten(root[""].children)
}

// buildGraph walks the vault and builds the complete link graph.
func buildGraph(vaultDir string) (*Graph, map[string][]string, map[string]string, error) {
	g := &Graph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	allPages := make(map[string]bool)
	pageTitles := make(map[string]string)

	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(vaultDir, path)
		pageID := strings.TrimSuffix(rel, ".md")
		allPages[pageID] = true
		if data, err := os.ReadFile(path); err == nil {
			pageTitles[pageID] = extractTitle(data)
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	err = filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(vaultDir, path)
		srcID := strings.TrimSuffix(rel, ".md")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		_, targets, _, _ := extractWikiLinks(data, rel)
		for _, tgt := range targets {
			g.Edges = append(g.Edges, GraphEdge{Source: srcID, Target: tgt})
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	added := make(map[string]bool)
	for id := range allPages {
		g.Nodes = append(g.Nodes, GraphNode{
			ID: id, Title: pageTitles[id], Path: id + ".html", Stub: false,
		})
		added[id] = true
	}
	for _, e := range g.Edges {
		if !added[e.Target] {
			g.Nodes = append(g.Nodes, GraphNode{
				ID: e.Target, Title: toHTMLName(e.Target), Path: e.Target + ".html", Stub: true,
			})
			added[e.Target] = true
		}
	}

	backlinks := make(map[string][]string)
	for _, e := range g.Edges {
		backlinks[e.Target] = append(backlinks[e.Target], e.Source)
	}
	bJSON, _ := json.Marshal(backlinks)
	os.WriteFile(filepath.Join(vaultDir, "..", "output", "backlinks.json"), bJSON, 0644)
	return g, backlinks, pageTitles, nil
}

func toHTMLName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".md")
}

// extractTitle extracts title from YAML frontmatter or first H1
func extractTitle(data []byte) string {
	if m := regexp.MustCompile(`(?m)^title:\s*(.+)\s*$`).FindSubmatch(data); len(m) > 0 {
		return strings.TrimSpace(string(m[1]))
	}
	if m := regexp.MustCompile(`(?m)^#\s+(.+)\s*$`).FindSubmatch(data); len(m) > 1 {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}

func pageIDFromRelPath(relPath string) string {
	return strings.TrimSuffix(relPath, ".md")
}

func downloadD3(graphDir string) {
	p := filepath.Join(graphDir, "d3.min.js")
	if _, err := os.Stat(p); err == nil {
		return
	}
	resp, err := http.Get("https://cdn.jsdelivr.net/npm/d3@7/dist/d3.min.js")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	os.WriteFile(p, body, 0644)
}

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	Links           []GraphRef `json:"links"`
	Backlinks       []GraphRef `json:"backlinks"`
	CurrentHref     string     `json:"currentHref"`
	Tags            []string   `json:"tags"`
	TableOfContents []TOCEntry `json:"tableOfContents"`
	Date            string     `json:"date"`
	ReadingTime     string     `json:"readingTime"`
}

// TOCEntry is a heading entry in the table of contents
type TOCEntry struct {
	Level int    `json:"level"` // 1 = h1, 2 = h2, etc.
	Text  string `json:"text"`
	ID    string `json:"id"`
}

// GraphRef is a reference to another page
type GraphRef struct {
	Title string `json:"title"`
	Href  string `json:"href"`
	Stub  bool   `json:"stub"`
}

// SearchEntry is a page entry in the search index
type SearchEntry struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	Content string `json:"content"` // plain text, stripped of markdown/html
}

// buildSearchIndex walks the vault and builds a search index of all pages.
func buildSearchIndex(vaultDir string) []SearchEntry {
	var entries []SearchEntry
	stripTags := regexp.MustCompile(`<[^>]*>`)
	stripMd := regexp.MustCompile(`[#*_~]`)
	stripMdAlt := regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	stripMdHdr := regexp.MustCompile(`^#{1,6}\s+`)
	stripMdHr := regexp.MustCompile(`^-{3,}\s*$`)
	stripMdCode := regexp.MustCompile("`[^`]+`")
	stripWs := regexp.MustCompile(`\s+`)

	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") || isIgnored(path) {
			return nil
		}
		relPath, _ := filepath.Rel(vaultDir, path)
		pageID := strings.TrimSuffix(relPath, ".md")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		data = removeFrontmatter(data)
		text := string(data)
		text = stripTags.ReplaceAllString(text, " ")
		text = stripMdCode.ReplaceAllString(text, "")
		text = stripMdHdr.ReplaceAllString(text, "")
		text = stripMdAlt.ReplaceAllString(text, "$1")
		text = stripMdHr.ReplaceAllString(text, " ")
		text = stripMd.ReplaceAllString(text, "")
		text = stripWs.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
		title := extractTitle(data)
		if title == "Untitled" {
			title = toHTMLName(pageID)
		}
		entries = append(entries, SearchEntry{
			Title:   title,
			Path:    pageID + ".html",
			Content: text,
		})
		return nil
	})
	return entries
}

// extractTOC extracts heading entries from rendered HTML for table of contents.
// Matches headings with auto-generated IDs like <h2 id="some-heading">Text</h2>
func extractTOC(htmlBody []byte) []TOCEntry {
	// Match <hN id="...">text</hN> — goldmark's auto-ID format
	re := regexp.MustCompile(`(?i)<h([1-6])\s+[^>]*id="([^"]+)"[^>]*>([^<]*)</h[1-6]>`)
	matches := re.FindAllSubmatch(htmlBody, -1)
	var toc []TOCEntry
	for _, m := range matches {
		if len(m) >= 4 {
			level, _ := strconv.Atoi(string(m[1]))
			id := string(m[2])
			text := string(m[3])
			toc = append(toc, TOCEntry{Level: level, Text: text, ID: id})
		}
	}
	return toc
}

// NavNode is a node in the navigation tree
type NavNode struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Href      string     `json:"href"`
	IndexHref string    `json:"indexHref,omitempty"`
	Children  []*NavNode `json:"children,omitempty"`
}

// wikiLinkRe matches [[Page]] and [[Page|Display Text]]
var wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// extractWikiLinks extracts wiki-style links from markdown content.
func extractWikiLinks(content []byte, sourceRelPath string) ([]byte, []string, []string) {
	var targets, rels []string
	result := wikiLinkRe.ReplaceAllFunc(content, func(match []byte) []byte {
		m := wikiLinkRe.FindSubmatch(match)
		if len(m) < 2 {
			return match
		}
		target := string(m[1])
		display := target
		if len(m) >= 3 && len(m[2]) > 0 {
			display = string(m[2])
		}
		targets = append(targets, target)

		srcDir := filepath.Dir(strings.TrimSuffix(sourceRelPath, ".md"))
		tgtBase := toHTMLName(target)
		tgtDir := filepath.Dir(target)
		rel, _ := filepath.Rel(srcDir, filepath.Join(tgtDir, tgtBase))
		rels = append(rels, rel)

		linkDisp := toHTMLName(target)
		if len(m) >= 3 && len(m[2]) > 0 {
			linkDisp = display
		}
		return []byte("[" + linkDisp + "](" + rel + ".html)")
	})
	return result, targets, rels
}

// computeRelHref computes relative href from source page to target page.
func computeRelHref(sourcePageID, targetPageID string) string {
	sourceDir := filepath.Dir(sourcePageID)
	rel, err := filepath.Rel(sourceDir, targetPageID)
	if err != nil {
		return targetPageID + ".html"
	}
	return rel + ".html"
}

// buildNavTree walks the vault and builds a hierarchical nav tree.
// "index" page is always first; folders sorted alphabetically; pages sorted alphabetically.
func buildNavTree(vaultDir string) []*NavNode {
	type entry struct{ pageID, title string }
	entries := []entry{}

	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") || isIgnored(path) {
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
		name      string
		path      string
		href      string
		indexHref string
		children  map[string]*tn
	}

	root := map[string]*tn{"": {name: "", path: "", href: "", indexHref: "", children: map[string]*tn{}}}

	for _, e := range entries {
		parts := strings.Split(e.pageID, "/")
		cur := root[""]
		for i, part := range parts {
			isLast := i == len(parts)-1
			// Detect if this part is an "index" page for the parent folder
			isIndexForParent := isLast && part == "index" && len(parts) > 1
			if cur.children[part] == nil {
				if isIndexForParent {
					// This page IS the index of cur (the parent folder)
					// Mark the parent folder with indexHref, don't create a separate child node
					cur.indexHref = e.pageID + ".html"
					// Use the index page's title as the folder name
					cur.name = e.title
				} else {
					child := &tn{
						name:      e.title,
						path:      e.pageID,
						href:      e.pageID + ".html",
						indexHref: "",
						children:  map[string]*tn{},
					}
					if !isLast {
						child.name = part
						child.path = ""
						child.href = ""
					}
					cur.children[part] = child
				}
			} else if isLast && isIndexForParent {
				// Folder already existed; mark its indexHref and use index title as folder name
				cur.indexHref = e.pageID + ".html"
				cur.name = e.title
			}
			if !isIndexForParent {
				cur = cur.children[part]
			}
		}
	}

	var flatten func(m map[string]*tn) []*NavNode
	flatten = func(m map[string]*tn) []*NavNode {
		var result []*NavNode
		for _, c := range m {
			node := &NavNode{Name: c.name, Path: c.path, Href: c.href, IndexHref: c.indexHref}
			if c.href == "" {
				node.Children = flatten(c.children)
			}
			result = append(result, node)
		}
		sort.Slice(result, func(i, j int) bool {
			aIsHome := result[i].Path == "index"
			bIsHome := result[j].Path == "index"
			if aIsHome != bIsHome {
				return aIsHome
			}
			aIsFolder := result[i].Href == ""
			bIsFolder := result[j].Href == ""
			if aIsFolder != bIsFolder {
				return aIsFolder
			}
			return result[i].Name < result[j].Name
		})
		return result
	}

	return flatten(root[""].children)
}

// buildGraph walks the vault and builds the complete link graph.
func buildGraph(vaultDir string) (*Graph, map[string][]string, map[string]string, error) {
	g := &Graph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	allPages := make(map[string]bool)
	pageTitles := make(map[string]string)

	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") || isIgnored(path) {
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
		if err != nil || !strings.HasSuffix(path, ".md") || isIgnored(path) {
			return nil
		}
		rel, _ := filepath.Rel(vaultDir, path)
		srcID := strings.TrimSuffix(rel, ".md")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		_, targets, _ := extractWikiLinks(data, rel)
		for _, tgt := range targets {
			// Skip wiki links to non-markdown files (e.g. [[diagram.drawio]])
			// The target is a path without extension — check if .md version exists in vault
			mdPath := filepath.Join(vaultDir, tgt+".md")
			if _, err := os.Stat(mdPath); os.IsNotExist(err) {
				continue
			}
			g.Edges = append(g.Edges, GraphEdge{Source: srcID, Target: tgt})
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	added := make(map[string]bool)
	for id := range allPages {
		title := pageTitles[id]
		// If page is an index page, use folder name only if frontmatter has no meaningful title
		if toHTMLName(id) == "index" {
			parts := strings.Split(id, "/")
			if len(parts) > 1 {
				// Only use folder name if title is empty or equals "index"
				if title == "" || title == "index" {
					title = parts[len(parts)-2]
				}
			}
		}
		g.Nodes = append(g.Nodes, GraphNode{
			ID: id, Title: title, Path: id + ".html", Stub: false,
		})
		added[id] = true
	}
	for _, e := range g.Edges {
		if !added[e.Target] && strings.HasSuffix(e.Target, ".md") {
			title := toHTMLName(e.Target)
			// If stub is an index page, use the folder name but preserve title case
			if title == "index" {
				parts := strings.Split(e.Target, "/")
				if len(parts) > 1 {
					title = parts[len(parts)-2]
				}
			}
			g.Nodes = append(g.Nodes, GraphNode{
				ID: e.Target, Title: title, Path: e.Target + ".html", Stub: true,
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

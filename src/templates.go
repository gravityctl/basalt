package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// generateHTMLTemplate produces the full HTML page for a rendered markdown file.
// navTreeJSON is the hierarchical navigation tree as JSON.
func generateHTMLTemplate(title string, htmlContent string, sourcePath string, pageGraph *PageGraph, navTreeJSON string) string {
	pageGraphJSON, _ := json.Marshal(pageGraph)

	css := `
	:root { --bg: #f8f8f8; --text: #333; --link: #2980b9; --sidebar-bg: #f0f0f0; --border: #e1e4e8; }
	* { box-sizing: border-box; }
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; margin: 0; background: var(--bg); color: var(--text); display: flex; min-height: 100vh; }
	/* 3-column layout: nav | content | graph */
	.layout { display: grid; grid-template-columns: 1fr 2fr 1fr; width: 100%; max-width: 100vw; align-items: start; }
	/* Left sidebar — nav */
	.sidebar-nav { background: var(--sidebar-bg); border-right: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-nav h2 { margin: 0 0 12px; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: #888; }
	.nav-tree { font-size: 0.9em; }
	.nav-folder { margin: 4px 0; }
	.nav-folder-header { cursor: pointer; padding: 4px 6px; border-radius: 4px; display: flex; align-items: center; gap: 4px; color: var(--text); user-select: none; }
	.nav-folder-header:hover { background: rgba(0,0,0,0.05); }
	.nav-folder-header .icon { font-size: 0.7em; transition: transform 0.15s; display: inline-block; }
	.nav-folder-header .icon.open { transform: rotate(90deg); }
	.nav-folder-children { padding-left: 16px; display: none; }
	.nav-folder-children.open { display: block; }
	.nav-page { padding: 4px 6px; border-radius: 4px; }
	.nav-page a { color: var(--link); text-decoration: none; font-weight: 400; }
	.nav-page a:hover { text-decoration: underline; }
	.nav-page.active a { font-weight: 600; color: #1a5f8a; }
	/* Center content */
	.content-col { padding: 20px; min-width: 0; }
	.content-col h1 { border-bottom: 1px solid var(--border); padding-bottom: 10px; margin: 0 0 20px; font-size: 1.5em; }
	.markdown-body { background: white; padding: 24px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
	.markdown-body p, .markdown-body li { font-size: 16px; }
	.markdown-body h2 { margin-top: 28px; }
	.markdown-body a { color: var(--link); text-decoration: none; font-weight: 500; }
	.markdown-body a:hover { text-decoration: underline; }
	.backlinks { margin-top: 24px; padding: 16px; background: white; border-radius: 6px; border: 1px solid var(--border); }
	.backlinks h3 { margin: 0 0 10px; font-size: 0.85em; text-transform: uppercase; letter-spacing: 0.05em; color: #888; }
	.backlinks ul { margin: 0; padding-left: 18px; font-size: 14px; }
	.backlinks li { margin: 4px 0; }
	/* Right sidebar — graph */
	.sidebar-graph { background: var(--sidebar-bg); border-left: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-graph h2 { margin: 0 0 12px; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: #888; }
	#local-graph { width: 100%; height: 200px; background: white; border: 1px solid var(--border); border-radius: 6px; }
	/* stub links */
	a.stub-link { color: #e67e22; font-style: italic; }
	`

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Basalt</title>
    <style>%s</style>
</head>
<body>
<div class="layout">
    <aside class="sidebar-nav">
        <h2>Nav</h2>
        <nav class="nav-tree" id="nav-tree"></nav>
    </aside>
    <main class="content-col">
        <h1>%s</h1>
        <div class="markdown-body">
            %s
        </div>
        %s
    </main>
    <aside class="sidebar-graph">
        <h2>Graph</h2>
        <div id="local-graph"></div>
    </aside>
</div>
<script>
window.pageGraphData = %s;
window.navTree = %s;
</script>
<script>
(function() {
    // d3.min.js is at OutputDir/graph/d3.min.js
    // Pages at OutputDir/pageID.html need ../graph/d3.min.js
    // Pages at OutputDir/subdir/pageID.html need ../../graph/d3.min.js
    var segs = location.pathname.split("/").filter(Boolean);
    var depth = Math.max(0, segs.length - 2);
    var d3Path = (depth >= 1 ? "../".repeat(depth) : "") + "graph/d3.min.js";
    var d3 = document.createElement("script");
    d3.src = d3Path;
    document.head.appendChild(d3);
})();
</script>
<script>
(function() {
    // --- Nav tree ---
    var navTree = window.navTree || [];
    function buildNavHTML(nodes, level) {
        var html = '';
        for (var i = 0; i < nodes.length; i++) {
            var node = nodes[i];
            if (node.children) {
                // Folder
                var folderId = 'folder-' + Math.random().toString(36).slice(2);
                html += '<div class="nav-folder">';
                html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                html += '<span class="icon">&#9654;</span> ' + escHtml(node.name);
                html += '</div>';
                html += '<div class="nav-folder-children" id="' + folderId + '">';
                html += buildNavHTML(node.children, level + 1);
                html += '</div>';
                html += '</div>';
            } else {
                // Page
                var active = (window.pageGraphData && node.href && location.pathname.endsWith(node.href.replace('../',''))) ? ' active' : '';
                html += '<div class="nav-page' + active + '"><a href="../' + node.href + '">' + escHtml(node.name) + '</a></div>';
            }
        }
        return html;
    }
    function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
    function toggleNavFolder(el) {
        var children = el.nextElementSibling;
        var icon = el.querySelector('.icon');
        children.classList.toggle('open');
        icon.classList.toggle('open');
    }
    document.getElementById('nav-tree').innerHTML = buildNavHTML(navTree, 0);

    // --- Per-page graph ---
    var container = document.getElementById('local-graph');
    if (!container || !window.pageGraphData) return;
    var data = window.pageGraphData;
    if (data.links.length === 0 && data.backlinks.length === 0) { container.style.display = 'none'; return; }
    var pageId = location.pathname.split('/').pop().replace('.html', '');
    var nodes = [{ id: pageId, title: document.title.replace(' - Basalt', ''), href: pageId + '.html', current: true }];
    var nodeIds = {};
    nodeIds[pageId] = true;
    data.links.forEach(function(l) { var id = l.href.replace('.html', ''); if (!nodeIds[id]) { nodes.push({ id: id, title: l.title, href: l.href, stub: l.stub }); nodeIds[id] = true; } });
    data.backlinks.forEach(function(bl) { var id = bl.href.replace('.html', ''); if (!nodeIds[id]) { nodes.push({ id: id, title: bl.title, href: bl.href }); nodeIds[id] = true; } });
    var edges = [];
    data.links.forEach(function(l) { edges.push({ source: pageId, target: l.href.replace('.html', '') }); });
    data.backlinks.forEach(function(bl) { edges.push({ source: bl.href.replace('.html', ''), target: pageId }); });
    var width = container.clientWidth || 180;
    var height = 200;
    var svg = d3.select(container).append('svg').attr('width', width).attr('height', height);
    var simulation = d3.forceSimulation(nodes)
        .force('link', d3.forceLink(edges).id(function(d) { return d.id; }).distance(50))
        .force('charge', d3.forceManyBody().strength(-120))
        .force('center', d3.forceCenter(width / 2, height / 2))
        .force('collision', d3.forceCollide().radius(20));
    var link = svg.append('g').selectAll('line').data(edges).enter().append('line').style('stroke', '#ccc').style('stroke-width', 1.5);
    var node = svg.append('g').selectAll('g').data(nodes).enter().append('g').attr('class', 'node')
        .call(d3.drag()
            .on('start', function(s) { if (!s.active) simulation.alphaTarget(0.3).restart(); s.subject.fx = s.subject.x; s.subject.fy = s.subject.y; })
            .on('drag', function(s) { s.subject.fx = s.x; s.subject.fy = s.y; })
            .on('end', function(s) { if (!s.active) simulation.alphaTarget(0); s.subject.fx = null; s.subject.fy = null; }));
    node.append('circle').attr('r', function(d) { return d.current ? 8 : 5; })
        .style('fill', function(d) { return d.stub ? '#e67e22' : (d.current ? '#2980b9' : '#3498db'); })
        .style('stroke', 'white').style('stroke-width', 1.5);
    node.append('text').attr('dx', 8).attr('dy', 3).style('font-size', '10px').style('fill', '#333').text(function(d) { return d.title; });
    node.on('click', function(event, d) { if (!d.stub && !d.current) window.location.href = d.href; });
    simulation.on('tick', function() {
        link.attr('x1', function(d) { return d.source.x; }).attr('y1', function(d) { return d.source.y; })
           .attr('x2', function(d) { return d.target.x; }).attr('y2', function(d) { return d.target.y; });
        node.attr('transform', function(d) { return 'translate(' + d.x + ',' + d.y + ')'; });
    });
})();
</script>
</body>
</html>`,
		title, css, title, htmlContent,
		buildBacklinksHTML(pageGraph),
		string(pageGraphJSON), navTreeJSON)
}

// buildBacklinksHTML renders the Links and Backlinks sections for a page
func buildBacklinksHTML(pg *PageGraph) string {
	if pg == nil || (len(pg.Links) == 0 && len(pg.Backlinks) == 0) {
		return ""
	}
	result := "<div class=\"backlinks\">"
	if len(pg.Links) > 0 {
		result += "<h3>Links</h3><ul>"
		for _, link := range pg.Links {
			classAttr := ""
			if link.Stub {
				classAttr = " class=\"stub-link\""
			}
			stubNote := map[bool]string{true: " (stub)"}[link.Stub]
			result += fmt.Sprintf("<li><a href=\"%s\"%s>%s</a>%s</li>", link.Href, classAttr, link.Title, stubNote)
		}
		result += "</ul>"
	}
	if len(pg.Backlinks) > 0 {
		if len(pg.Links) > 0 {
			result += "<h3>Backlinks</h3><ul>"
		} else {
			result += "<h3>Backlinks</h3><ul>"
		}
		for _, bl := range pg.Backlinks {
			result += fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", bl.Href, bl.Title)
		}
		result += "</ul>"
	}
	result += "</div>"
	return result
}

// generateStubHTML creates a placeholder page for a dead link target
func generateStubHTML(pageID string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s — Create Page</title>
    <style>
        :root { --bg: #f8f8f8; --text: #333; --link: #2980b9; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; margin: 0; background: var(--bg); color: var(--text); display: flex; min-height: 100vh; }
        .layout { display: grid; grid-template-columns: 1fr 2fr 1fr; width: 100%%; }
        .content-col { padding: 20px; }
        .stub { background: #fff3cd; border: 1px solid #ffc107; padding: 20px; border-radius: 6px; }
        .stub h2 { margin-top: 0; color: #856404; }
        code { background: #f8f8f8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
<div class="layout">
    <main class="content-col">
        <h1>%s</h1>
        <div class="stub">
            <h2>📄 Page Not Found</h2>
            <p>This page doesn't exist yet. To create it, add a file named <code>%s.md</code> to your vault.</p>
        </div>
    </main>
</div>
</body>
</html>`, pageID, pageID, pageID)
}

// writeGraphViewer writes the full vault D3 graph viewer.
func writeGraphViewer(graphDir string, graphJSON []byte) {
	downloadD3(graphDir)
	writeFullGraphViewer(graphDir, graphJSON)
}

func writeFullGraphViewer(graphDir string, graphJSON []byte) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Graph View — Basalt</title>
    <style>
        :root { --bg: #f8f8f8; --text: #333; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; background: var(--bg); color: var(--text); }
        h1 { padding: 20px; margin: 0; font-size: 1.2em; border-bottom: 1px solid #e1e4e8; background: white; }
        #graph { width: 100vw; height: calc(100vh - 61px); }
        .node { cursor: pointer; }
        .node circle { fill: #2980b9; stroke: white; stroke-width: 2px; }
        .node.stub circle { fill: #e67e22; stroke: #fff; }
        .node text { font-size: 12px; fill: #333; pointer-events: none; }
        .link { stroke: #ccc; stroke-width: 1.5px; }
        .link:hover { stroke: #2980b9; }
        #legend { position: absolute; top: 70px; right: 20px; background: white; padding: 15px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 0.85em; }
        #legend h3 { margin: 0 0 10px; }
        #legend span { display: inline-block; width: 12px; height: 12px; border-radius: 50%%; margin-right: 6px; vertical-align: middle; }
        .legend-page { background: #2980b9; }
        .legend-stub { background: #e67e22; }
    </style>
</head>
<body>
    <h1>📊 Vault Graph View</h1>
    <div id="legend">
        <h3>Legend</h3>
        <div><span class="legend-page"></span>Page</div>
        <div><span class="legend-stub"></span>Stub (dead link)</div>
    </div>
    <div id="graph"></div>
    <script src="d3.min.js"></script>
    <script>
    var graph = %s;
    var width = document.getElementById("graph").clientWidth;
    var height = document.getElementById("graph").clientHeight;
    var svg = d3.select("#graph").append("svg").attr("width", width).attr("height", height);
    var simulation = d3.forceSimulation(graph.nodes)
        .force("link", d3.forceLink(graph.edges).id(function(d) { return d.id; }).distance(80))
        .force("charge", d3.forceManyBody().strength(-200))
        .force("center", d3.forceCenter(width / 2, height / 2))
        .force("collision", d3.forceCollide().radius(30));
    var link = svg.append("g").selectAll("line").data(graph.edges).enter().append("line").attr("class", "link");
    var node = svg.append("g").selectAll("g").data(graph.nodes).enter().append("g").attr("class", function(d) { return "node" + (d.stub ? " stub" : ""); })
        .call(d3.drag()
            .on("start", function(e) { if (!e.active) simulation.alphaTarget(0.3).restart(); e.subject.fx = e.subject.x; e.subject.fy = e.subject.y; })
            .on("drag", function(e) { e.subject.fx = e.x; e.subject.fy = e.y; })
            .on("end", function(e) { if (!e.active) simulation.alphaTarget(0); e.subject.fx = null; e.subject.fy = null; }));
    node.append("circle").attr("r", 8);
    node.append("text").attr("dx", 12).attr("dy", 4).text(function(d) { return d.title; });
    node.on("click", function(event, d) { if (!d.stub) window.location.href = "../" + d.path; });
    simulation.on("tick", function() {
        link.attr("x1", function(d) { return d.source.x; }).attr("y1", function(d) { return d.source.y; })
           .attr("x2", function(d) { return d.target.x; }).attr("y2", function(d) { return d.target.y; });
        node.attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });
    });
    </script>
</body>
</html>`
	os.WriteFile(filepath.Join(graphDir, "index.html"), []byte(fmt.Sprintf(html, graphJSON)), 0644)
}

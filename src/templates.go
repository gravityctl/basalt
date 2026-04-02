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
	backlinksHTML := buildBacklinksHTML(pageGraph)
	tagsHTML := buildTagsHTML(pageGraph)
	tocHTML := buildTocHTML(pageGraph)

	css := `
	/* Dark mode (default) */
	:root, [data-theme="dark"] { --bg: #1e1e1e; --text: #e0e0e0; --link: #6bb3d9; --sidebar-bg: #252525; --border: #3a3a3a; --heading: #ffffff; --muted: #888888; --card-bg: #2a2a2a; }
	/* Light mode */
	[data-theme="light"] { --bg: #f8f8f8; --text: #333; --link: #2980b9; --sidebar-bg: #f0f0f0; --border: #e1e4e8; --heading: #1a1a1a; --muted: #888888; --card-bg: #ffffff; }
	* { box-sizing: border-box; }
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; margin: 0; background: var(--bg); color: var(--text); display: flex; min-height: 100vh; }
	.layout { display: grid; grid-template-columns: 1fr 2fr 1fr; width: 100%; max-width: 100vw; align-items: start; }
	/* Left sidebar — nav */
	.sidebar-nav { background: var(--sidebar-bg); border-right: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-nav h2 { margin: 0 0 12px; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.nav-tree { font-size: 0.9em; }
	.nav-folder { margin: 4px 0; }
	.nav-folder-header { cursor: pointer; padding: 4px 6px; border-radius: 4px; display: flex; align-items: center; gap: 4px; color: var(--text); user-select: none; }
	.nav-folder-header:hover { background: rgba(255,255,255,0.05); }
	[data-theme="light"] .nav-folder-header:hover { background: rgba(0,0,0,0.05); }
	.nav-folder-header .icon { font-size: 0.7em; transition: transform 0.15s; display: inline-block; }
	.nav-folder-header .icon.open { transform: rotate(90deg); }
	.nav-folder-children { padding-left: 16px; display: none; }
	.nav-folder-children.open { display: block; }
	.nav-page { padding: 4px 6px; border-radius: 4px; }
	.nav-page a, .nav-folder-header a { color: var(--link); text-decoration: none; font-weight: 400; }
	.nav-page a:visited, .nav-folder-header a:visited { color: var(--link); }
	.nav-page a:hover, .nav-folder-header a:hover { text-decoration: underline; }
	.nav-page.active a { font-weight: 700; text-decoration: underline; color: var(--link); }
	/* Center content */
	.content-col { padding: 20px; min-width: 0; }
	.content-col h1 { border-bottom: 1px solid var(--border); padding-bottom: 10px; margin: 0 0 6px; font-size: 1.5em; color: var(--heading); }
	.page-meta { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; font-size: 0.8em; color: var(--muted); }
	.page-meta-left { font-style: italic; }
	.page-meta-right { font-style: normal; }
	.markdown-body { background: var(--card-bg); padding: 24px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
	.markdown-body p, .markdown-body li { font-size: 16px; }
	.markdown-body h2 { margin-top: 28px; color: var(--heading); }
	.markdown-body h3 { color: var(--heading); }
	.markdown-body a { color: var(--link); text-decoration: none; font-weight: 500; }
	.markdown-body a:hover { text-decoration: underline; }
	/* Right sidebar */
	.sidebar-right { background: var(--sidebar-bg); border-left: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-right h2 { margin: 0 0 12px; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	#local-graph { width: 100%; height: 180px; background: var(--card-bg); border: 1px solid var(--border); border-radius: 6px; margin-bottom: 16px; }
	.sidebar-section { margin-bottom: 16px; }
	.sidebar-section h3 { margin: 0 0 8px; font-size: 0.75em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.sidebar-links { background: var(--card-bg); border: 1px solid var(--border); border-radius: 6px; padding: 12px; font-size: 0.85em; }
	.sidebar-links ul { margin: 0; padding-left: 16px; }
	.sidebar-links li { margin: 4px 0; }
	.sidebar-links a { color: var(--link); text-decoration: none; font-weight: 500; }
	.sidebar-links a:hover { text-decoration: underline; }
	a.stub-link { color: #e67e22; font-style: italic; }
	.tags { margin-top: 16px; display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
	.tags-label { font-size: 0.8em; color: var(--muted); margin-right: 4px; }
	.tag { display: inline-block; padding: 2px 8px; background: var(--link); color: var(--bg); border-radius: 12px; font-size: 0.8em; font-weight: 500; opacity: 0.85; }
	/* Table of contents */
	.toc { margin-top: 16px; font-size: 0.85em; }
	.toc h3 { margin: 0 0 8px; font-size: 0.75em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.toc-list { list-style: none; margin: 0; padding: 0; }
	.toc-item { margin: 2px 0; }
	.toc-item a { color: var(--link); text-decoration: none; }
	.toc-item a:hover { text-decoration: underline; }
	.toc-item.level-1 { padding-left: 0; }
	.toc-item.level-2 { padding-left: 12px; }
	.toc-item.level-3 { padding-left: 24px; }
	.toc-item.level-4 { padding-left: 36px; }
	.toc-item.level-5 { padding-left: 48px; }
	.toc-item.level-6 { padding-left: 60px; }
	/* Theme toggle */
	.site-name { font-size: 0.9em; font-weight: 700; color: var(--heading); margin-bottom: 12px; padding: 0 6px; }
	.sidebar-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
	.sidebar-header h2 { margin: 0; }
	.theme-toggle { background: none; border: none; color: var(--muted); cursor: pointer; padding: 0; font-size: 1.2em; line-height: 1; display: flex; align-items: center; justify-content: center; width: 24px; height: 24px; }
	.theme-toggle:hover { color: var(--text); }
	.theme-toggle svg { width: 1em; height: 1em; }
	`

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Basalt</title>
    <style>%s</style>
</head>
<body>
<div class="layout">
    <aside class="sidebar-nav">
        <div class="site-name">Basalt</div>
        <div class="sidebar-header">
            <h2>Browse</h2>
            <button class="theme-toggle" id="theme-toggle" title="Toggle dark/light mode">&#9788;</button>
        </div>
        <nav class="nav-tree" id="nav-tree"></nav>
    </aside>
    <main class="content-col">
        <h1>%s</h1>
        <div class="page-meta">
            <span class="page-meta-left">%s</span>
            <span class="page-meta-right">%s</span>
        </div>
        <div class="markdown-body">
            %s
        </div>
    </main>
    <aside class="sidebar-right">
        <h2>Graph</h2>
        <div id="local-graph"></div>
        %s
        %s
        %s
    </aside>
</div>
<script>
window.pageGraphData = %s;
window.navTree = %s;
</script>
<script>
// ---- Nav: render immediately ----
(function() {
    function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
    function toggleNav(el) {
        var children = el.nextElementSibling;
        var icon = el.querySelector('.icon');
        children.classList.toggle('open');
        icon.classList.toggle('open');
    }
    window.toggleNavFolder = toggleNav;
    function buildNavHTML(nodes) {
        var html = '';
        for (var i = 0; i < nodes.length; i++) {
            var node = nodes[i];
            var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
            var prefix = depth > 0 ? '../'.repeat(depth) : '';
            if (node.children) {
                var fid = 'f-' + Math.random().toString(36).slice(2);
                var folderLabel = escHtml(node.name);
                if (node.indexHref) {
                    var folderLink = '<a href="' + prefix + node.indexHref + '" onclick="event.stopPropagation()">' + folderLabel + '</a>';
                    html += '<div class="nav-folder">';
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="icon">&#9654;</span> ' + folderLink;
                    html += '</div>';
                } else {
                    html += '<div class="nav-folder">';
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="icon">&#9654;</span> ' + folderLabel;
                    html += '</div>';
                }
                html += '<div class="nav-folder-children" id="' + fid + '">';
                html += buildNavHTML(node.children);
                html += '</div></div>';
            } else {
                var href = prefix + node.href;
                var isActive = window.pageGraphData && window.pageGraphData.currentHref && window.pageGraphData.currentHref === node.href;
                var cls = isActive ? 'nav-page active' : 'nav-page';
                html += '<div class="' + cls + '"><a href="' + href + '">' + escHtml(node.name) + '</a></div>';
            }
        }
        return html;
    }
    var navEl = document.getElementById('nav-tree');
    if (navEl) navEl.innerHTML = buildNavHTML(window.navTree || []);
})();
</script>
<script>
// ---- Theme toggle ----
(function() {
    var html = document.documentElement;
    var toggle = document.getElementById('theme-toggle');
    // Apply saved preference or default to dark
    var saved = localStorage.getItem('basalt-theme');
    if (saved) { html.setAttribute('data-theme', saved); }
    else { html.setAttribute('data-theme', 'dark'); }
    updateIcon();
    toggle.addEventListener('click', function() {
        var current = html.getAttribute('data-theme');
        var next = current === 'dark' ? 'light' : 'dark';
        html.setAttribute('data-theme', next);
        localStorage.setItem('basalt-theme', next);
        updateIcon();
    });
    function updateIcon() {
        var isDark = html.getAttribute('data-theme') === 'dark';
        // Inline SVG icons that use currentColor (matches text color)
        if (isDark) {
            toggle.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>';
            toggle.title = 'Switch to light mode';
        } else {
            toggle.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>';
            toggle.title = 'Switch to dark mode';
        }
    }
})();
</script>
<script>
// ---- D3 graph: load async and draw ----
(function() {
    var _cur = window.pageGraphData && window.pageGraphData.currentHref ? window.pageGraphData.currentHref : "";
    var _dp = _cur.split('/').length - 1;
    var _d3p = _dp > 0 ? '../'.repeat(_dp) + 'graph/d3.min.js' : 'graph/d3.min.js';
    function drawGraph() {
        try {
        console.log('graph: drawGraph called');
        var container = document.getElementById('local-graph');
        if (!container) { console.log('graph: no container'); return; }
        if (typeof window.d3 === 'undefined') { console.log('graph: d3 not loaded'); return; }
        var _d3 = window.d3;
        console.log('graph: d3 version=' + (_d3.version || 'unknown'));
        console.log('graph: d3 forceSimulation=' + typeof _d3.forceSimulation);
        var data = window.pageGraphData;
        var pageId = location.pathname.split('/').filter(Boolean).pop().replace('.html', '');
        var nodes = [{ id: pageId, title: document.title.replace(' - Basalt', ''), href: location.pathname.split('/').filter(Boolean).pop(), current: true }];
        var nodeIds = {};
        nodeIds[pageId] = true;
        data.links.forEach(function(l) { var id = l.href.replace('.html',''); if (!nodeIds[id]) { nodes.push({ id: id, title: l.title, href: l.href, stub: l.stub }); nodeIds[id] = true; } });
        data.backlinks.forEach(function(bl) { var id = bl.href.replace('.html',''); if (!nodeIds[id]) { nodes.push({ id: id, title: bl.title, href: bl.href }); nodeIds[id] = true; } });
        var edges = [];
        data.links.forEach(function(l) { edges.push({ source: pageId, target: l.href.replace('.html','') }); });
        data.backlinks.forEach(function(bl) { edges.push({ source: bl.href.replace('.html',''), target: pageId }); });
        var w = container.clientWidth || 180;
        var h = 180;
        var svg = _d3.select(container).append('svg').attr('width', w).attr('height', h);
        // Create SVG groups BEFORE simulation starts so tick can update them
        console.log('graph: svg created, w=' + w + ', h=' + h);
        console.log('graph: nodes count=' + nodes.length + ', edges count=' + edges.length);
        var linkG = svg.append('g');
        var nodeG = svg.append('g');
        console.log('graph: groups created, linkG type=' + typeof linkG + ', nodeG type=' + typeof nodeG);
        var sim = _d3.forceSimulation(nodes)
            .force('link', _d3.forceLink(edges).id(function(d) { return d.id; }).distance(40))
            .force('charge', _d3.forceManyBody().strength(-80))
            .force('center', _d3.forceCenter(w / 2, h / 2))
            .force('collision', _d3.forceCollide().radius(15));
        // Render nodes/links immediately (before sim ticks)
        var link = linkG.selectAll('line').data(edges).enter().append('line').style('stroke', '#ccc').style('stroke-width', 1.5);
        var node = nodeG.selectAll('g').data(nodes).enter().append('g')
            .style('cursor', function(d) { return d.stub || d.current ? 'default' : 'pointer'; });
        node.call(_d3.drag()
            .on('start', function(e) { if (!e.active) sim.alphaTarget(0.3).restart(); e.subject.fx = e.subject.x; e.subject.fy = e.subject.y; })
            .on('drag', function(e) { e.subject.fx = e.x; e.subject.fy = e.y; })
            .on('end', function(e) { if (!e.active) sim.alphaTarget(0); e.subject.fx = null; e.subject.fy = null; }));
        node.on('click', function(e, d) { if (!d.stub && !d.current) window.location.href = d.href; });
        node.append('circle').attr('r', function(d) { return d.current ? 7 : 4 }).style('fill', function(d) { return d.stub ? '#e67e22' : (d.current ? '#2980b9' : '#3498db'); }).style('stroke', 'white').style('stroke-width', 1.5);
        node.append('text').attr('dx', 7).attr('dy', 3).style('font-size', '9px').style('fill', 'currentColor').style('opacity', '0.8').text(function(d) { return d.title; });
        console.log('graph: sim created, node count=' + nodes.length);
        console.log('graph: link selection=' + (typeof link) + ', node selection=' + (typeof node));
        console.log('graph: calling tick...');
        // Update positions on every tick
        sim.on('tick', function() {
            try {
            link.attr('x1', function(d) { return d.source.x; }).attr('y1', function(d) { return d.source.y; })
              .attr('x2', function(d) { return d.target.x; }).attr('y2', function(d) { return d.target.y; });
            node.attr('transform', function(d) { return 'translate(' + d.x + ',' + d.y + ')'; });
            } catch(e) { console.log('graph: tick error=' + e); }
        });
        console.log('graph: tick registered, simulation should be running');
        } catch(e) { console.log('graph: drawGraph error: ' + e); }
    }
    var s = document.createElement("script");
    s.src = _d3p;
    s.onload = function() { console.log('graph: script loaded'); drawGraph(); };
    s.onerror = function() { console.log('graph: script failed to load: ' + _d3p); };
    document.head.appendChild(s);
})();
</script>
</body>
</html>`,
		title, css, title,
		pageGraph.ReadingTime,
		pageGraph.Date,
		htmlContent,
		backlinksHTML,
		tagsHTML,
		tocHTML,
		string(pageGraphJSON), navTreeJSON)
}

// buildBacklinksHTML renders Links and Backlinks for the sidebar
func buildBacklinksHTML(pg *PageGraph) string {
	if pg == nil || (len(pg.Links) == 0 && len(pg.Backlinks) == 0) {
		return ""
	}
	s := "<div class=\"sidebar-section\"><div class=\"sidebar-links\">"
	if len(pg.Links) > 0 {
		s += "<h3>Links</h3><ul>"
		for _, l := range pg.Links {
		 cls := map[bool]string{true: " class=\"stub-link\""}[l.Stub]
		 s += fmt.Sprintf("<li><a href=\"%s\"%s>%s</a>%s</li>", l.Href, cls, l.Title, map[bool]string{true: " *(stub)"}[l.Stub])
		}
		s += "</ul>"
	}
	if len(pg.Backlinks) > 0 {
		s += "<h3>Backlinks</h3><ul>"
		for _, bl := range pg.Backlinks {
		 s += fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", bl.Href, bl.Title)
		}
		s += "</ul>"
	}
	s += "</div></div>"
	return s
}

// buildTagsHTML renders the tags section for a page
func buildTagsHTML(pg *PageGraph) string {
	if pg == nil || len(pg.Tags) == 0 {
		return ""
	}
	s := "<div class=\"tags\"><span class=\"tags-label\">Tags:</span>"
	for _, tag := range pg.Tags {
		s += fmt.Sprintf("<span class=\"tag\">%s</span>", tag)
	}
	s += "</div>"
	return s
}

// buildTocHTML renders the table of contents for the sidebar
func buildTocHTML(pg *PageGraph) string {
	if pg == nil || len(pg.TableOfContents) == 0 {
		return ""
	}
	s := "<div class=\"toc\"><h3>On this page</h3><ul class=\"toc-list\">"
	for _, entry := range pg.TableOfContents {
		s += fmt.Sprintf("<li class=\"toc-item level-%d\"><a href=\"#%s\">%s</a></li>", entry.Level, entry.ID, entry.Text)
	}
	s += "</ul></div>"
	return s
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
        .node text { font-size: 12px; fill: currentColor; opacity: 0.8; pointer-events: none; }
        .link { stroke: #ccc; stroke-width: 1.5px; }
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
    var w = document.getElementById("graph").clientWidth;
    var h = document.getElementById("graph").clientHeight;
    var svg = d3.select("#graph").append("svg").attr("width", w).attr("height", h);
    var sim = d3.forceSimulation(graph.nodes)
        .force("link", d3.forceLink(graph.edges).id(function(d) { return d.id; }).distance(80))
        .force("charge", d3.forceManyBody().strength(-200))
        .force("center", d3.forceCenter(w / 2, h / 2))
        .force("collision", d3.forceCollide().radius(30));
    sim.on("tick", function() {
        svg.selectAll("line").attr("x1", function(d) { return d.source.x; }).attr("y1", function(d) { return d.source.y; })
          .attr("x2", function(d) { return d.target.x; }).attr("y2", function(d) { return d.target.y; });
        svg.selectAll("g").attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });
    });
    sim.on("end", function() {
        var link = svg.append("g").selectAll("line").data(graph.edges).enter().append("line").attr("class", "link");
        var node = svg.append("g").selectAll("g").data(graph.nodes).enter().append("g").attr("class", function(d) { return "node" + (d.stub ? " stub" : ""); })
            .call(d3.drag()
                .on("start", function(e) { if (!e.active) sim.alphaTarget(0.3).restart(); e.subject.fx = e.subject.x; e.subject.fy = e.subject.y; })
                .on("drag", function(e) { e.subject.fx = e.x; e.subject.fy = e.y; })
                .on("end", function(e) { if (!e.active) sim.alphaTarget(0); e.subject.fx = null; e.subject.fy = null; }))
            .on("click", function(event, d) { if (!d.stub) window.location.href = "../" + d.path; });
        node.append("circle").attr("r", 8);
        node.append("text").attr("dx", 12).attr("dy", 4).text(function(d) { return d.title; });
    });
    </script>
</body>
</html>`
	os.WriteFile(filepath.Join(graphDir, "index.html"), []byte(fmt.Sprintf(html, graphJSON)), 0644)
}

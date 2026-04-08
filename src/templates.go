package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// generateHTMLTemplate produces the full HTML page for a rendered markdown file.
// navTreeJSON is the hierarchical navigation tree as JSON.
func generateHTMLTemplate(title string, htmlContent string, sourcePath string, pageGraph *PageGraph, navTreeJSON string, siteCfg SiteConfig) string {
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
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; margin: 0; background: var(--bg); color: var(--text); }
	.layout { display: grid; grid-template-columns: 1fr 2fr 1fr; width: 100%; max-width: 100vw; align-items: start; }
	/* Mobile nav toggle */
	.mobile-nav-toggle { display: none; }
	.mobile-header { display: none; }
	@media (max-width: 768px) {
		.mobile-nav-toggle { display: block; background: var(--sidebar-bg); border: 1px solid var(--border); color: var(--text); border-radius: 6px; padding: 8px 12px; font-size: 1.2em; cursor: pointer; }
		.mobile-header { position: fixed; top: 0; left: 0; right: 0; z-index: 998; display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: var(--sidebar-bg); border-bottom: 1px solid var(--border); }
		.mobile-header .mobile-site-name { flex: 1; font-size: 1em; font-weight: 600; color: var(--heading); margin: 0; padding: 0; border: none; }
		.layout { display: block; }
		.sidebar-nav {
			display: none;
			position: fixed; top: 0; left: 0; right: 0; height: 100vh; width: 100vw; z-index: 1000;
			transform: translateX(-100%); transition: transform 0.25s ease;
			box-shadow: none;
		}
		.sidebar-nav.open { display: block; transform: translateX(0); }
		.sidebar-nav.closed { display: block; transform: translateX(-100%); }
		.content-col { padding: 16px 20px; }
		.sidebar-right { display: none; }
		.sidebar-right .sidebar-section { margin-bottom: 8px; }
	}
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
	.markdown-body img { max-width: 100%; height: auto; display: block; margin: 0 auto; }
	/* Right sidebar */
	.sidebar-right { background: var(--sidebar-bg); border-left: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-right h2 { margin: 0; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.graph-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
	.graph-header button { background: none; border: none; color: var(--muted); cursor: pointer; font-size: 0.9em; padding: 0; line-height: 1; }
	.graph-header button:hover { color: var(--text); }
	.close-nav { display: none; background: none; border: none; color: var(--muted); cursor: pointer; font-size: 1em; padding: 0; }
	@media (max-width: 768px) {
		.close-nav { display: block; }
	}
	.full-graph-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.7); z-index: 1000; display: flex; align-items: center; justify-content: center; }
	.full-graph-modal { background: var(--sidebar-bg); border: 1px solid var(--border); border-radius: 8px; width: 90vw; height: 85vh; display: flex; flex-direction: column; overflow: hidden; }
	.full-graph-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid var(--border); }
	.full-graph-header h2 { margin: 0; font-size: 0.9em; color: var(--heading); text-transform: uppercase; letter-spacing: 0.05em; }
	.full-graph-header button { background: none; border: none; color: var(--muted); cursor: pointer; font-size: 1.2em; padding: 0; line-height: 1; }
	.full-graph-header button:hover { color: var(--text); }
	#full-graph-container { flex: 1; overflow: hidden; }
	#full-graph-container iframe { width: 100%; height: 100%; border: none; background: var(--bg); }
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
	.site-name { border-bottom: 1px solid var(--border); padding-bottom: 10px; margin: 0 0 12px; font-size: 1.5em; font-weight: 700; color: var(--heading); padding-left: 6px; }
	.sidebar-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
	.search-bar { width: 100%; background: var(--card-bg); border: 1px solid var(--border); color: var(--muted); cursor: pointer; padding: 6px 10px; border-radius: 4px; font-size: 0.85em; text-align: left; margin-bottom: 12px; display: flex; align-items: center; justify-content: space-between; }
	.search-bar .icon { font-size: 2em; }
	.search-bar:hover { border-color: var(--link); color: var(--text); }
	.search-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.7); z-index: 1000; display: flex; align-items: flex-start; justify-content: center; padding-top: 10vh; }
	.search-modal { background: var(--sidebar-bg); border: 1px solid var(--border); border-radius: 8px; width: 90vw; max-width: 600px; max-height: 80vh; display: flex; flex-direction: column; overflow: hidden; }
	.search-header { display: flex; align-items: center; border-bottom: 1px solid var(--border); padding: 12px 16px; gap: 12px; }
	#search-input { flex: 1; background: none; border: none; color: var(--text); font-size: 1em; outline: none; }
	#search-input::placeholder { color: var(--muted); }
	#close-search { background: none; border: none; color: var(--muted); cursor: pointer; font-size: 1.2em; padding: 0; line-height: 1; }
	#close-search:hover { color: var(--text); }
	#search-results { overflow-y: auto; padding: 8px; }
	.search-result { display: block; padding: 10px 12px; border-radius: 4px; text-decoration: none; color: var(--text); }
	.search-result:hover { background: var(--card-bg); }
	.search-result-title { font-weight: 600; margin-bottom: 4px; color: var(--heading); }
	.search-result-snippet { font-size: 0.8em; color: var(--muted); line-height: 1.4; }
	.search-result-snippet mark { background: rgba(255,220,50,0.3); color: inherit; border-radius: 2px; }
	.search-empty { padding: 20px; text-align: center; color: var(--muted); font-size: 0.9em; }
	.sidebar-header h2 { margin: 0; }
	.theme-toggle { background: none; border: none; color: var(--muted); cursor: pointer; padding: 0; font-size: 1.2em; line-height: 1; display: flex; align-items: center; justify-content: center; width: 24px; height: 24px; }
	.theme-toggle:hover { color: var(--text); }
	.theme-toggle svg { width: 1em; height: 1em; }
	`

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="%[13]s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%[1]s - %[12]s</title>
    <style>%[2]s</style>
</head>
<body>
    <div class="mobile-header">
        <button id="mobile-nav-toggle" class="mobile-nav-toggle" aria-label="Toggle navigation">☰</button>
        <span class="mobile-site-name">%[12]s</span>
    </div>
<div class="layout">
    <aside class="sidebar-nav">
        <div class="site-name">%[12]s</div>
        <div class="sidebar-header">
            <button id="close-nav" class="close-nav" aria-label="Close navigation">✕</button>
            <h2>Browse</h2>
            <button class="theme-toggle" id="theme-toggle" title="Toggle dark/light mode">&#9788;</button>
        </div>
        <button id="open-search" class="search-bar" type="button">Search <span class="icon">&#8981;</span></button>
        <nav class="nav-tree" id="nav-tree"></nav>
    </aside>
    <main class="content-col">
        <h1>%[1]s</h1>
        <div class="page-meta">
            <span class="page-meta-left">%[4]s</span>
            <span class="page-meta-right">%[5]s</span>
        </div>
        <div class="markdown-body">
            %[6]s
        </div>
    </main>
    <aside class="sidebar-right">
        <div class="graph-header">
            <h2>Graph</h2>
            <button id="open-full-graph" title="Full vault graph" aria-label="Open full vault graph">⤢</button>
        </div>
        <div id="local-graph"></div>
        %[7]s
        %[8]s
        %[9]s
    </aside>
</div>
<script>
window.siteName = "%[12]s";
    // Mobile nav toggle
    var navToggle = document.getElementById('mobile-nav-toggle');
    var closeNav = document.getElementById('close-nav');
    var sidebarNav = document.querySelector('.sidebar-nav');
    if (navToggle) {
        navToggle.addEventListener('click', function() { sidebarNav.classList.toggle('open'); sidebarNav.classList.remove('closed'); });
    }
    if (closeNav) {
        closeNav.addEventListener('click', function() { sidebarNav.classList.remove('open'); sidebarNav.classList.add('closed'); });
    }
window.siteTheme = "%[13]s";
window.pageGraphData = %[10]s;
window.navTree = %[11]s;
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
    window.toggleNavFolder = function(el) {
        var children = el.nextElementSibling;
        var icon = el.querySelector('.icon');
        var fid = children.id;
        children.classList.toggle('open');
        icon.classList.toggle('open');
        var expanded = getExpandedFolders();
        if (children.classList.contains('open')) {
            if (expanded.indexOf(fid) < 0) expanded.push(fid);
        } else {
            expanded = expanded.filter(function(f) { return f !== fid; });
        }
        saveExpandedFolders(expanded);
    };
    function buildNavHTML(nodes, parentPath) {
        var html = '';
        var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
        var baseDepth = depth;
        var prefix = depth > 0 ? '../'.repeat(depth) : '';
        var expandedFolders = getExpandedFolders();
        for (var i = 0; i < nodes.length; i++) {
            var node = nodes[i];
            if (node.children) {
                var folderId = 'navf-' + (parentPath ? parentPath + '-' : '') + node.name;
                var isOpen = expandedFolders.indexOf(folderId) >= 0;
                var folderLabel = escHtml(node.name);
                var iconClass = isOpen ? 'icon open' : 'icon';
                html += '<div class="nav-folder">';
                if (node.indexHref) {
                    var folderLink = '<a href="' + prefix + node.indexHref + '" onclick="event.stopPropagation()">' + folderLabel + '</a>';
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="' + iconClass + '">&#9654;</span> ' + folderLink;
                    html += '</div>';
                } else {
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="' + iconClass + '">&#9654;</span> ' + folderLabel;
                    html += '</div>';
                }
                html += '<div class="nav-folder-children' + (isOpen ? ' open' : '') + '" id="' + folderId + '">';
                html += buildNavHTML(node.children, folderId);
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
    function getExpandedFolders() {
        try { return JSON.parse(sessionStorage.getItem('basalt-nav-open') || []); } catch(e) { return []; }
    }
    function saveExpandedFolders(folders) {
        try { sessionStorage.setItem('basalt-nav-open', JSON.stringify(folders)); } catch(e) {}
    }
    var navEl = document.getElementById('nav-tree');
    if (navEl) navEl.innerHTML = buildNavHTML(window.navTree || [], '');
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
        var pageId = _cur.replace('.html', '');
        var nodes = [{ id: pageId, title: document.title.replace(' - ' + window.siteName, ''), href: _cur, current: true }];
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
<div id="full-graph-overlay" class="full-graph-overlay" style="display:none;">
    <div class="full-graph-modal">
        <div class="full-graph-header">
            <h2>Full Vault Graph</h2>
            <button id="close-full-graph" aria-label="Close">&times;</button>
        </div>
        <div id="full-graph-container"></div>
    </div>
</div>
<script>
// ---- Full vault graph modal ----
(function() {
    var overlay = document.getElementById('full-graph-overlay');
    var container = document.getElementById('full-graph-container');
    var openBtn = document.getElementById('open-full-graph');
    var closeBtn = document.getElementById('close-full-graph');

    openBtn.addEventListener('click', function() {
        // Compute path to graph/index.html from current page
        var segs = window.pageGraphData.currentHref.split('/').filter(Boolean);
        var depth = Math.max(0, segs.length - 1);
        var base = depth > 0 ? '../'.repeat(depth) : '';
        var graphPath = base + 'graph/index.html';
        container.innerHTML = '<iframe src="' + graphPath + '" style="width:100%;height:100%;border:none;"></iframe>';
        overlay.style.display = 'flex';
    });

    closeBtn.addEventListener('click', function() {
        overlay.style.display = 'none';
        container.innerHTML = '';
    });

    overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
            overlay.style.display = 'none';
            container.innerHTML = '';
        }
    });
})();
</script>
<div id="search-overlay" class="search-overlay" style="display:none;">
    <div class="search-modal">
        <div class="search-header">
            <input id="search-input" type="text" placeholder="Search pages..." autocomplete="off" />
            <button id="close-search" aria-label="Close">&times;</button>
        </div>
        <div id="search-results"></div>
    </div>
</div>
<script>
// ---- Search modal ----
(function() {
    var overlay = document.getElementById('search-overlay');
    var input = document.getElementById('search-input');
    var results = document.getElementById('search-results');
    var openBtn = document.getElementById('open-search');
    var closeBtn = document.getElementById('close-search');
    var searchIndex = null;

    function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

    function highlight(text, term) {
        if (!term) return escHtml(text);
        var idx = text.toLowerCase().indexOf(term.toLowerCase());
        if (idx < 0) return escHtml(text.slice(0, 200));
        var start = Math.max(0, idx - 60);
        var end = Math.min(text.length, idx + term.length + 120);
        var snippet = (start > 0 ? '...' : '') + text.slice(start, end) + (end < text.length ? '...' : '');
        var re = new RegExp(escHtml(term).replace(/[-\\^$*+?.()|[\]{}]/g, '\\$&'), 'gi');
        return escHtml(snippet).replace(re, function(m) { return '<mark>' + m + '</mark>'; });
    }

    function doSearch(term) {
        if (!searchIndex) return;
        var q = term.toLowerCase();
        var matches = [];
        for (var i = 0; i < searchIndex.length; i++) {
            var e = searchIndex[i];
            var score = 0;
            if (e.title.toLowerCase().indexOf(q) >= 0) score += 10;
            if (e.content.toLowerCase().indexOf(q) >= 0) score += 1;
            if (score > 0) matches.push({ entry: e, score: score });
        }
        matches.sort(function(a, b) { return b.score - a.score; });
        if (matches.length === 0 || term.length === 0) {
            results.innerHTML = '<div class="search-empty">Start typing to search...</div>';
            return;
        }
        var html = '';
        for (var j = 0; j < Math.min(matches.length, 20); j++) {
            var m = matches[j];
            var e = m.entry;
            // Compute depth for relative path
            var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
            var prefix = depth > 0 ? '../'.repeat(depth) : '';
            html += '<a class="search-result" href="' + prefix + e.path + '">';
            html += '<div class="search-result-title">' + escHtml(e.title) + '</div>';
            html += '<div class="search-result-snippet">' + highlight(e.content, term) + '</div>';
            html += '</a>';
        }
        results.innerHTML = html;
    }

    openBtn.addEventListener('click', function() {
        overlay.style.display = 'flex';
        input.focus();
        if (!searchIndex) {
            var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
            var prefix = depth > 0 ? '../'.repeat(depth) : '';
            fetch(prefix + 'search.json').then(function(r) { return r.json(); }).then(function(data) {
                searchIndex = data;
                doSearch(input.value);
            }).catch(function() { searchIndex = []; });
        }
    });

    input.addEventListener('input', function() { doSearch(input.value); });

    closeBtn.addEventListener('click', function() {
        overlay.style.display = 'none';
        input.value = '';
        results.innerHTML = '';
    });

    overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
            overlay.style.display = 'none';
            input.value = '';
            results.innerHTML = '';
        }
    });

    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && overlay.style.display === 'flex') {
            overlay.style.display = 'none';
            input.value = '';
            results.innerHTML = '';
        }
    });
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
		string(pageGraphJSON), navTreeJSON,
		siteCfg.SiteName, siteCfg.SiteTheme)
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
<button id="mobile-nav-toggle" class="mobile-nav-toggle" aria-label="Toggle navigation">☰</button>
<div class="layout">
    <main class="content-col">
        <h1>%[1]s</h1>
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
func writeGraphViewer(graphDir string, graphJSON []byte, siteTheme string, siteName string) {
	downloadD3(graphDir)
	writeFullGraphViewer(graphDir, graphJSON, siteTheme, siteName)
}

func writeFullGraphViewer(graphDir string, graphJSON []byte, siteTheme string, siteName string) {
	html := `<!DOCTYPE html>
<html lang="en" data-theme="%s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Graph View — %s</title>
    <style>
        :root, [data-theme="dark"] { --bg: #1e1e1e; --text: #e0e0e0; --border: #3a3a3a; --heading: #ffffff; --card-bg: #2a2a2a; --link: #6bb3d9; }
        [data-theme="light"] { --bg: #f8f8f8; --text: #333; --border: #e1e4e8; --heading: #1a1a1a; --card-bg: #ffffff; --link: #2980b9; }
        html, body { overflow: hidden; height: 100%%; margin: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: var(--bg); color: var(--text); }
        #graph { width: 100vw; height: 100vh; overflow: hidden; }
        .node { cursor: pointer; }
        .node circle { fill: var(--link); stroke: white; stroke-width: 2px; }
        .node.stub circle { fill: #e67e22; stroke: #fff; }
        .node text { font-size: 12px; fill: currentColor; opacity: 0; pointer-events: none; transition: opacity 0.2s; }
        .node.hovered text, .node.neighbor text { opacity: 1; }
        .link { stroke: var(--link); stroke-width: 2px; transition: stroke-opacity 0.2s; }
        .node.dimmed circle { opacity: 0.2; }
        .node.dimmed text { opacity: 0; }
        .link.dimmed { stroke-opacity: 0.5; }
        #legend { position: absolute; top: 20px; right: 20px; background: var(--card-bg); padding: 15px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.2); font-size: 0.85em; border: 1px solid var(--border); }
        #legend h3 { margin: 0 0 10px; color: var(--heading); }
        #legend span { display: inline-block; width: 12px; height: 12px; border-radius: 50%%; margin-right: 6px; vertical-align: middle; }
        .legend-page { background: var(--link); }
        .legend-stub { background: #e67e22; }
    </style>
</head>
<body>
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
    // Zoom/pan via scroll wheel and drag on SVG background
    var zoomG = svg.append("g");
    svg.call(d3.zoom().scaleExtent([0.1, 4]).on("zoom", function(e) { zoomG.attr("transform", e.transform); }));
    var sim = d3.forceSimulation(graph.nodes)
        .force("link", d3.forceLink(graph.edges).id(function(d) { return d.id; }).distance(40))
        .force("charge", d3.forceManyBody().strength(0))
        .force("center", d3.forceCenter(w / 2, h / 2))
        .force("collision", d3.forceCollide().radius(20))
        .alpha(0.3);
    var link = zoomG.selectAll("line").data(graph.edges).enter().append("line").attr("class", "link");
    // Build neighbor set for hover highlighting (edges are still strings here)
    var neighborOf = {};
    graph.nodes.forEach(function(n) { neighborOf[n.id] = new Set(); });
    graph.edges.forEach(function(e) {
        var sid = typeof e.source === 'object' ? e.source.id : e.source;
        var tid = typeof e.target === 'object' ? e.target.id : e.target;
        neighborOf[sid].add(tid);
        neighborOf[tid].add(sid);
    });
    var node = zoomG.selectAll("g").data(graph.nodes).enter().append("g").attr("class", function(d) { return "node" + (d.stub ? " stub" : ""); })
        .call(d3.drag()
            .on("start", function(e) { if (!e.active) sim.alphaTarget(0.3).restart(); e.subject.fx = e.subject.x; e.subject.fy = e.subject.y; })
            .on("drag", function(e) { e.subject.fx = e.x; e.subject.fy = e.y; })
            .on("end", function(e) { if (!e.active) sim.alphaTarget(0); e.subject.fx = null; e.subject.fy = null; }))
        .on("mouseover", function(event, d) {
            var nid = d.id;
            var neighbors = neighborOf[nid] || new Set();
            node.classed("hovered", function(n) { return n.id === nid; });
            node.classed("neighbor", function(n) { return n.id !== nid && neighbors.has(n.id); });
            node.classed("dimmed", function(n) { return n.id !== nid && !neighbors.has(n.id); });
            link.classed("dimmed", function(l) {
                var sid = l.source.id || l.source;
                var tid = l.target.id || l.target;
                return sid !== nid && tid !== nid;
            });
        })
        .on("mouseout", function() {
            node.classed("hovered", false).classed("neighbor", false).classed("dimmed", false);
            link.classed("dimmed", false);
        })
        .on("click", function(event, d) { if (!d.stub) { sim.stop(); graph.nodes.forEach(function(n) { n.fx = n.x; n.fy = n.y; }); window.location.href = "../" + d.path; } });
    node.append("circle").attr("r", 8);
    node.append("text").attr("dx", 12).attr("dy", 4).text(function(d) { return d.title; });
    sim.on("tick", function() {
        link.attr("x1", function(d) { return d.source.x; }).attr("y1", function(d) { return d.source.y; })
          .attr("x2", function(d) { return d.target.x; }).attr("y2", function(d) { return d.target.y; });
        node.attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });
    });
    </script>
</body>
</html>`
	os.WriteFile(filepath.Join(graphDir, "index.html"), []byte(fmt.Sprintf(html, siteTheme, siteName, graphJSON)), 0644)
}

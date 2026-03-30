package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// generateHTMLTemplate produces the full HTML page for a rendered markdown file.
func generateHTMLTemplate(title string, htmlContent string, sourcePath string, pageGraph *PageGraph) string {
	css := `
	:root { --bg: #f8f8f8; --text: #333; --link: #2980b9; }
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; max-width: 850px; margin: 0 auto; padding: 20px; background: var(--bg); color: var(--text); }
	h1 { border-bottom: 1px solid #e1e4e8; padding-bottom: 10px; }
	a { color: var(--link); text-decoration: none; font-weight: 500; }
	a:hover { text-decoration: underline; }
	a.stub-link { color: #e67e22; font-style: italic; }
	nav { margin-bottom: 20px; font-size: 0.85em; color: #666; }
	nav a { color: #666; }
	.content { margin-top: 20px; }
	.markdown-body { background: white; padding: 30px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
	p, li { font-size: 16px; }
	h2 { margin-top: 30px; }
	#local-graph { height: 300px; border: 1px solid #e1e4e8; border-radius: 6px; margin-top: 20px; background: white; }
	.backlinks { margin-top: 30px; padding: 15px; background: white; border-radius: 6px; border: 1px solid #e1e4e8; }
	.backlinks h3 { margin-top: 0; font-size: 0.9em; color: #666; }
	.backlinks ul { margin: 0; padding-left: 20px; }
	.backlinks li { font-size: 14px; }
	`

	navHTML := ""
	if sourcePath != "" {
		navHTML = fmt.Sprintf("<nav><a href=\"../graph/index.html\">📊 Graph View</a> — Source: <code>%s</code></nav>", sourcePath)
	}

	backlinksHTML := buildBacklinksHTML(pageGraph)

	pageGraphJSON, _ := json.Marshal(pageGraph)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Basalt</title>
    <style>%s</style>
</head>
<body>
    <header>
        <h1>%s</h1>
        %s
    </header>
    <main class="content">
        <div class="markdown-body">
            %s
        </div>
        %s
    <script>
    (function() {
        var container = document.getElementById("local-graph");
        if (!container || !window.pageGraphData) return;
        var data = window.pageGraphData;
        if (data.links.length === 0 && data.backlinks.length === 0) { container.style.display = "none"; return; }
        var pageId = location.pathname.split("/").pop().replace(".html", "");
        var nodes = [{ id: pageId, title: document.title.replace(" - Basalt", ""), current: true }];
        var nodeIds = new Set([pageId]);
        data.links.forEach(function(l) { var id = l.href.replace(".html", ""); if (!nodeIds.has(id)) { nodes.push({ id: id, title: l.title, stub: l.stub }); nodeIds.add(id); } });
        data.backlinks.forEach(function(bl) { var id = bl.href.replace(".html", ""); if (!nodeIds.has(id)) { nodes.push({ id: id, title: bl.title }); nodeIds.add(id); } });
        var edges = [];
        data.links.forEach(function(l) { edges.push({ source: pageId, target: l.href.replace(".html", "") }); });
        data.backlinks.forEach(function(bl) { edges.push({ source: bl.href.replace(".html", ""), target: pageId }); });
        var width = container.clientWidth, height = 300;
        var svg = d3.select(container).append("svg").attr("width", width).attr("height", height);
        var simulation = d3.forceSimulation(nodes)
            .force("link", d3.forceLink(edges).id(function(d) { return d.id; }).distance(60))
            .force("charge", d3.forceManyBody().strength(-150))
            .force("center", d3.forceCenter(width / 2, height / 2))
            .force("collision", d3.forceCollide().radius(25));
        var link = svg.append("g").selectAll("line").data(edges).enter().append("line").style("stroke", "#ccc").style("stroke-width", 1.5);
        var node = svg.append("g").selectAll("g").data(nodes).enter().append("g").attr("class", "node")
            .call(d3.drag()
                .on("start", function(s) { if (!s.active) simulation.alphaTarget(0.3).restart(); s.subject.fx = s.subject.x; s.subject.fy = s.subject.y; })
                .on("drag", function(s) { s.subject.fx = s.x; s.subject.fy = s.y; })
                .on("end", function(s) { if (!s.active) simulation.alphaTarget(0); s.subject.fx = null; s.subject.fy = null; }));
        node.append("circle").attr("r", function(d) { return d.current ? 10 : 6; })
            .style("fill", function(d) { return d.stub ? "#e67e22" : (d.current ? "#2980b9" : "#3498db"); })
            .style("stroke", "white").style("stroke-width", 2);
        node.append("text").attr("dx", 10).attr("dy", 4).style("font-size", "11px").style("fill", "#333").text(function(d) { return d.title; });
        node.on("click", function(event, d) { if (!d.stub && !d.current) window.location.href = d.id + ".html"; });
        node.on("mouseover", function(event, d) { link.style("stroke", function(l) { return (l.source.id === d.id || l.target.id === d.id) ? "#2980b9" : "#ccc"; }); });
        node.on("mouseout", function() { link.style("stroke", "#ccc"); });
        simulation.on("tick", function() {
            link.attr("x1", function(d) { return d.source.x; }).attr("y1", function(d) { return d.source.y; }).attr("x2", function(d) { return d.target.x; }).attr("y2", function(d) { return d.target.y; });
            node.attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });
        });
    })();
    </script>
</body>
</html>`, title, css, title, navHTML, htmlContent, backlinksHTML, string(pageGraphJSON))
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
		result += "<h3>Backlinks</h3><ul>"
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
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; max-width: 850px; margin: 0 auto; padding: 20px; background: var(--bg); color: var(--text); }
        .stub { background: #fff3cd; border: 1px solid #ffc107; padding: 20px; border-radius: 6px; margin-top: 40px; }
        .stub h2 { margin-top: 0; color: #856404; }
        code { background: #f8f8f8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <header>
        <h1>%s</h1>
    </header>
    <main>
        <div class="stub">
            <h2>📄 Page Not Found</h2>
            <p>This page doesn't exist yet. To create it, add a file named <code>%s.md</code> to your vault.</p>
        </div>
    </main>
</body>
</html>`, pageID, pageID, pageID)
}

// writeGraphViewer writes the full vault D3 graph viewer and per-page graph script.
// graphJSON is embedded inline so the page works without a server (no CORS/fetch issues).
func writeGraphViewer(graphDir string, graphJSON []byte) {
	writeFullGraphViewer(graphDir, graphJSON)
	writeLocalGraphScript(graphDir)
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
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script>
    // Embedded graph data — avoids CORS issues when opening via file://
    const graph = %s;
    renderGraph(graph);

    function renderGraph(graph) {
        const width = document.getElementById("graph").clientWidth;
        const height = document.getElementById("graph").clientHeight;

        const svg = d3.select("#graph").append("svg")
            .attr("width", width)
            .attr("height", height);

        const simulation = d3.forceSimulation(graph.nodes)
            .force("link", d3.forceLink(graph.edges).id(d => d.id).distance(80))
            .force("charge", d3.forceManyBody().strength(-200))
            .force("center", d3.forceCenter(width / 2, height / 2))
            .force("collision", d3.forceCollide().radius(30));

        const link = svg.append("g")
            .selectAll("line")
            .data(graph.edges)
            .enter().append("line")
            .attr("class", "link");

        const node = svg.append("g")
            .selectAll("g")
            .data(graph.nodes)
            .enter().append("g")
            .attr("class", d => "node" + (d.stub ? " stub" : ""))
            .call(d3.drag()
                .on("start", dragstarted)
                .on("drag", dragged)
                .on("end", dragended));

        node.append("circle").attr("r", 8);
        node.append("text").attr("dx", 12).attr("dy", 4).text(d => d.title);

        node.on("click", (event, d) => {
            if (!d.stub) window.location.href = "../" + d.path;
        });

        node.on("mouseover", function(event, d) {
            link.style("stroke", l => (l.source.id === d.id || l.target.id === d.id) ? "#2980b9" : "#ccc");
            link.style("stroke-width", l => (l.source.id === d.id || l.target.id === d.id) ? 3 : 1.5);
        });

        node.on("mouseout", function() {
            link.style("stroke", "#ccc").style("stroke-width", 1.5);
        });

        simulation.on("tick", () => {
            link
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);
            node.attr("transform", d => "translate(" + d.x + "," + d.y + ")");
        });

        function dragstarted(event) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            event.subject.fx = event.subject.x;
            event.subject.fy = event.subject.y;
        }
        function dragged(event) {
            event.subject.fx = event.x;
            event.subject.fy = event.y;
        }
        function dragended(event) {
            if (!event.active) simulation.alphaTarget(0);
            event.subject.fx = null;
            event.subject.fy = null;
        }
    }
    </script>
</body>
</html>`
	os.WriteFile(filepath.Join(graphDir, "index.html"), []byte(fmt.Sprintf(html, graphJSON)), 0644)
}

func writeLocalGraphScript(graphDir string) {
	js := `// graph-local.js — per-page local graph using D3 force-directed layout
(function() {
    const container = document.getElementById('local-graph');
    if (!container || !window.pageGraphData) return;

    const data = window.pageGraphData;
    if (data.links.length === 0 && data.backlinks.length === 0) {
        container.style.display = 'none';
        return;
    }

    const pageId = location.pathname.split('/').pop().replace('.html', '');
    const nodes = [{ id: pageId, title: document.title.replace(' - Basalt', ''), current: true }];
    const nodeIds = new Set([pageId]);

    data.links.forEach(l => {
        const id = l.href.replace('.html', '');
        if (!nodeIds.has(id)) {
            nodes.push({ id, title: l.title, stub: l.stub });
            nodeIds.add(id);
        }
    });

    data.backlinks.forEach(bl => {
        const id = bl.href.replace('.html', '');
        if (!nodeIds.has(id)) {
            nodes.push({ id, title: bl.title });
            nodeIds.add(id);
        }
    });

    const edges = [];
    data.links.forEach(l => edges.push({ source: pageId, target: l.href.replace('.html', '') }));
    data.backlinks.forEach(bl => edges.push({ source: bl.href.replace('.html', ''), target: pageId }));

    const width = container.clientWidth;
    const height = 300;

    const svg = d3.select(container).append('svg').attr('width', width).attr('height', height);

    const simulation = d3.forceSimulation(nodes)
        .force('link', d3.forceLink(edges).id(d => d.id).distance(60))
        .force('charge', d3.forceManyBody().strength(-150))
        .force('center', d3.forceCenter(width / 2, height / 2))
        .force('collision', d3.forceCollide().radius(25));

    const link = svg.append('g').selectAll('line')
        .data(edges).enter().append('line')
        .style('stroke', '#ccc').style('stroke-width', 1.5);

    const node = svg.append('g').selectAll('g')
        .data(nodes).enter().append('g')
        .attr('class', 'node')
        .call(d3.drag()
            .on('start', s => { if (!s.active) simulation.alphaTarget(0.3).restart(); s.subject.fx = s.subject.x; s.subject.fy = s.subject.y; })
            .on('drag', s => { s.subject.fx = s.x; s.subject.fy = s.y; })
            .on('end', s => { if (!s.active) simulation.alphaTarget(0); s.subject.fx = null; s.subject.fy = null; }));

    node.append('circle')
        .attr('r', d => d.current ? 10 : 6)
        .style('fill', d => d.stub ? '#e67e22' : (d.current ? '#2980b9' : '#3498db'))
        .style('stroke', 'white').style('stroke-width', 2);

    node.append('text').attr('dx', 10).attr('dy', 4).style('font-size', '11px').style('fill', '#333').text(d => d.title);

    node.on('click', (event, d) => { if (!d.stub && !d.current) window.location.href = d.id + '.html'; });
    node.on('mouseover', function(event, d) { link.style('stroke', l => (l.source.id === d.id || l.target.id === d.id) ? '#2980b9' : '#ccc'); });
    node.on('mouseout', function() { link.style('stroke', '#ccc'); });

    simulation.on('tick', () => {
        link.attr('x1', d => d.source.x).attr('y1', d => d.source.y).attr('x2', d => d.target.x).attr('y2', d => d.target.y);
        node.attr('transform', d => 'translate(' + d.x + ',' + d.y + ')');
    });
})();
`
	os.WriteFile(filepath.Join(graphDir, "graph-local.js"), []byte(js), 0644)
}

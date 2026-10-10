package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FlowNode is a changed symbol (or a file fallback) in a change-flow graph.
type FlowNode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Layer  string `json:"layer"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Status string `json:"status"`
}

// FlowEdge connects two nodes in the inferred request/data path.
type FlowEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

type flowResponse struct {
	Scope string     `json:"scope"`
	ID    string     `json:"id,omitempty"`
	Nodes []FlowNode `json:"nodes"`
	Edges []FlowEdge `json:"edges"`
}

type flowSource struct {
	path   string
	status string
	lines  []int
	added  bool
}

var (
	flowRoutePattern = regexp.MustCompile(`(?i)(?:\.get|\.post|\.put|\.patch|\.delete|\.all)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
	flowFetchPattern = regexp.MustCompile(`(?i)(?:fetch|axios\.(?:get|post|put|patch|delete))\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
	flowWordPattern  = regexp.MustCompile(`[A-Za-z_$][\w$]*`)
	flowSQLTable     = regexp.MustCompile(`(?i)create\s+table\s+(?:if\s+not\s+exists\s+)?["'\x60]?([\w.]+)`)
)

func flowLayer(rel string) string {
	p := strings.ToLower(filepath.ToSlash(rel))
	base := filepath.Base(p)
	ext := strings.ToLower(filepath.Ext(p))
	if strings.Contains(p, "/api/") || strings.HasPrefix(p, "api/") ||
		strings.Contains(p, "/routes/") || strings.HasPrefix(p, "routes/") ||
		strings.Contains(p, "/handlers/") || strings.HasPrefix(p, "handlers/") ||
		strings.Contains(p, "/controllers/") || strings.HasPrefix(p, "controllers/") ||
		strings.HasPrefix(base, "server.") {
		return "api"
	}
	if strings.Contains(p, "/migrations/") || strings.HasPrefix(p, "migrations/") ||
		strings.Contains(p, "/prisma/") || strings.HasPrefix(p, "prisma/") ||
		strings.Contains(p, "/models/") || strings.HasPrefix(p, "models/") ||
		strings.HasPrefix(base, "schema.") || ext == ".sql" || ext == ".json" {
		return "db"
	}
	if ext == ".tsx" || ext == ".jsx" || ext == ".vue" || ext == ".svelte" || ext == ".astro" ||
		strings.Contains(p, "/components/") || strings.HasPrefix(p, "components/") ||
		(strings.Contains(p, "/pages/") || strings.HasPrefix(p, "pages/")) {
		return "ui"
	}
	return "service"
}

func flowStatus(code string) string {
	if code == "A" || code == "?" || code == "U" {
		return "added"
	}
	if code == "D" {
		return "deleted"
	}
	return "modified"
}

func uniqueFlowInts(values ...[]int) []int {
	seen := make(map[int]bool)
	var out []int
	for _, list := range values {
		for _, n := range list {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Ints(out)
	return out
}

func flowSymbolsForHunks(symbols []Symbol, marks []int, lineCount int) []Symbol {
	if len(symbols) == 0 || len(marks) == 0 {
		return nil
	}
	selected := make([]Symbol, 0, len(symbols))
	for i, sym := range symbols {
		start := sym.Line
		end := lineCount
		if i+1 < len(symbols) {
			end = symbols[i+1].Line - 1
		}
		if end < start {
			end = start
		}
		for _, line := range marks {
			if line >= start && line <= end {
				selected = append(selected, sym)
				break
			}
		}
	}
	return selected
}

func (s *Server) flowSources(r *http.Request, scope string) ([]flowSource, string, int, string) {
	root := s.ix.Root()
	files := map[string]string{}
	id := ""
	switch scope {
	case "working":
		files = gitStatusAgainst(root, "HEAD")
		for _, rel := range gitUntracked(root) {
			files[rel] = "?"
		}
	case "pr":
		if s.pr == nil || s.diffBase == "" || s.prHeadSHA == "" {
			return nil, "", http.StatusNotFound, "not a PR review session"
		}
		files = gitFilesBetween(root, s.diffBase, s.prHeadSHA)
	case "thread":
		id = strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			return nil, "", http.StatusBadRequest, "missing thread id"
		}
		t := s.threads.Get(id)
		if t == nil {
			return nil, "", http.StatusNotFound, errThreadGone.Error()
		}
		turn := len(t.Turns) - 1
		if n := r.URL.Query().Get("turn"); n != "" {
			parsed, err := strconv.Atoi(n)
			if err != nil || parsed < 1 || parsed > len(t.Turns) {
				return nil, "", http.StatusBadRequest, "bad thread turn"
			}
			turn = parsed - 1
		} else {
			// The plan request itself is a turn too. Prefer the newest finished
			// turn that actually touched files so planning doesn't hide the change
			// map it is meant to compare against.
			for i := len(t.Turns) - 1; i >= 0; i-- {
				if !t.Turns[i].Running && t.Turns[i].Tracked && len(t.Turns[i].Changed) > 0 {
					turn = i
					break
				}
			}
		}
		if turn < 0 {
			return nil, "", http.StatusNotFound, "thread has no turns"
		}
		tr := t.Turns[turn]
		if !tr.Tracked {
			return nil, "", http.StatusConflict, "this thread turn has no tracked file changes"
		}
		workingFiles := gitStatusAgainst(root, "HEAD")
		for _, rel := range tr.Changed {
			if _, _, ok := s.safePath(rel); ok {
				rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
				status := workingFiles[rel]
				if status == "" {
					status = "M"
				}
				files[rel] = status
			}
		}
	default:
		return nil, "", http.StatusBadRequest, "scope must be working, pr, or thread"
	}

	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	if len(paths) > 200 {
		paths = paths[:200]
	}
	sources := make([]flowSource, 0, len(paths))
	untracked := map[string]bool{}
	if scope != "pr" {
		for _, rel := range gitUntracked(root) {
			untracked[rel] = true
		}
	}
	for _, rel := range paths {
		var added, modified, deleted []int
		if scope == "pr" {
			added, modified, deleted = parseDiffHunks(gitDiffBetween(root, rel, s.diffBase, s.prHeadSHA))
		} else {
			added, modified, deleted = gitHunksAgainst(root, rel, "HEAD")
		}
		sources = append(sources, flowSource{path: rel, status: flowStatus(files[rel]), lines: uniqueFlowInts(added, modified, deleted), added: untracked[rel] || files[rel] == "A"})
	}
	return sources, id, 0, ""
}

func (s *Server) buildFlow(ctx context.Context, sources []flowSource, scope, id string) flowResponse {
	root := s.ix.Root()
	response := flowResponse{Scope: scope, ID: id, Nodes: []FlowNode{}, Edges: []FlowEdge{}}
	nodeSources := map[string]flowSource{}
	nodeIDs := map[string]bool{}
	for _, src := range sources {
		abs, rel, ok := s.safePath(src.path)
		if !ok {
			continue
		}
		body, err := os.ReadFile(abs)
		if err != nil {
			// Deleted files have no current source to outline. Keep one file node so
			// removals remain visible in the graph.
			if src.status == "deleted" {
				label := filepath.Base(rel)
				node := FlowNode{ID: rel + "#file", Label: label, Layer: flowLayer(rel), Path: rel, Status: "deleted"}
				response.Nodes = append(response.Nodes, node)
				nodeSources[node.ID] = src
				nodeIDs[node.ID] = true
			}
			continue
		}
		symbols, _ := Outline(abs, rel)
		if strings.EqualFold(filepath.Ext(rel), ".sql") {
			for line, text := range strings.Split(string(body), "\n") {
				if match := flowSQLTable.FindStringSubmatch(text); match != nil {
					symbols = append(symbols, Symbol{Name: filepath.Base(match[1]), Kind: "table", Line: line + 1})
				}
			}
			sort.Slice(symbols, func(i, j int) bool { return symbols[i].Line < symbols[j].Line })
		}
		lineCount := strings.Count(string(body), "\n") + 1
		changedSymbols := flowSymbolsForHunks(symbols, src.lines, lineCount)
		if src.added && len(symbols) > 0 {
			changedSymbols = symbols
		}
		status := src.status
		if src.added {
			status = "added"
		}
		if len(changedSymbols) == 0 {
			label := filepath.Base(rel)
			id := rel + "#file"
			node := FlowNode{ID: id, Label: label, Layer: flowLayer(rel), Path: rel, Status: status}
			response.Nodes = append(response.Nodes, node)
			nodeSources[id] = src
			nodeIDs[id] = true
			continue
		}
		for _, sym := range changedSymbols {
			nodeID := rel + "#" + sym.Name
			if nodeIDs[nodeID] {
				nodeID += "@" + strconv.Itoa(sym.Line)
			}
			node := FlowNode{ID: nodeID, Label: sym.Name, Layer: flowLayer(rel), Path: rel, Line: sym.Line, Status: status}
			response.Nodes = append(response.Nodes, node)
			nodeSources[nodeID] = src
			nodeIDs[nodeID] = true
		}
	}

	// Infer call edges from references between changed symbols. This inexpensive
	// text pass is deliberately a heuristic; LSP results below refine it when a
	// server is already available.
	byName := map[string][]FlowNode{}
	for _, n := range response.Nodes {
		byName[n.Label] = append(byName[n.Label], n)
	}
	edges := map[string]FlowEdge{}
	addEdge := func(from, to, kind string) {
		if from == to {
			return
		}
		key := from + "\x00" + to + "\x00" + kind
		edges[key] = FlowEdge{From: from, To: to, Kind: kind}
	}
	routeNodes := map[string][]FlowNode{}
	for _, n := range response.Nodes {
		if n.Layer != "api" {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(n.Path))
		b, _ := os.ReadFile(abs)
		for _, match := range flowRoutePattern.FindAllSubmatch(b, 8) {
			routeNodes[strings.TrimSpace(string(match[1]))] = append(routeNodes[strings.TrimSpace(string(match[1]))], n)
		}
		if strings.Contains(strings.ToLower(n.Path), "/api/") || strings.HasPrefix(strings.ToLower(n.Path), "api/") {
			route := "/" + strings.TrimPrefix(n.Path[strings.Index(strings.ToLower(n.Path), "api/"):], "")
			route = strings.TrimSuffix(route, filepath.Ext(route))
			routeNodes[route] = append(routeNodes[route], n)
		}
	}
	for _, from := range response.Nodes {
		abs := filepath.Join(root, filepath.FromSlash(from.Path))
		body, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		words := map[string]bool{}
		for _, word := range flowWordPattern.FindAll(body, -1) {
			words[string(word)] = true
		}
		for name, targets := range byName {
			if name == from.Label || len(name) < 3 {
				continue
			}
			if words[name] {
				for _, to := range targets {
					if from.Path == to.Path && from.Line >= to.Line {
						continue
					}
					addEdge(from.ID, to.ID, "call")
				}
			}
		}
		for _, match := range flowFetchPattern.FindAllSubmatch(body, 16) {
			url := string(match[1])
			for route, targets := range routeNodes {
				if route != "" && strings.Contains(url, route) {
					for _, to := range targets {
						addEdge(from.ID, to.ID, "fetch")
					}
				}
			}
		}
		for _, to := range response.Nodes {
			if to.Layer != "db" || to.Path == from.Path || len(to.Label) < 3 {
				continue
			}
			if words[to.Label] {
				addEdge(from.ID, to.ID, "query")
			}
		}
	}

	// Query one outgoing call level only for LSP servers that are already ready;
	// graph generation never starts a language server as a side effect.
	deadline, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
	defer cancel()
	for _, n := range response.Nodes {
		if deadline.Err() != nil {
			break
		}
		if n.Status == "deleted" || n.Line < 1 {
			continue
		}
		state, _ := s.lsp.State(n.Path)
		if state != lspReady {
			continue
		}
		abs, _, ok := s.safePath(n.Path)
		if !ok {
			continue
		}
		roots, err := s.lsp.PrepareCalls(deadline, abs, n.Path, n.Line-1, 0)
		if err != nil {
			continue
		}
		for _, item := range roots {
			if item.Name != n.Label {
				continue
			}
			calls, err := s.lsp.Calls(deadline, n.Path, item.Item, true)
			if err != nil {
				continue
			}
			for _, callee := range calls {
				if callee.Ext {
					continue
				}
				matched := false
				for _, target := range byName[callee.Name] {
					if target.Path == callee.Path {
						addEdge(n.ID, target.ID, "call")
						matched = true
					}
				}
				if !matched {
					id := callee.Path + "#" + callee.Name
					if _, exists := nodeSources[id]; !exists {
						contextNode := FlowNode{ID: id, Label: callee.Name, Layer: flowLayer(callee.Path), Path: callee.Path, Line: callee.Line, Status: "context"}
						response.Nodes = append(response.Nodes, contextNode)
						byName[callee.Name] = append(byName[callee.Name], contextNode)
						nodeSources[id] = flowSource{path: callee.Path, status: "context"}
						nodeIDs[id] = true
					}
					addEdge(n.ID, id, "call")
				}
			}
			break
		}
	}
	for _, edge := range edges {
		response.Edges = append(response.Edges, edge)
	}
	sort.Slice(response.Nodes, func(i, j int) bool {
		if response.Nodes[i].Layer != response.Nodes[j].Layer {
			return layerOrder(response.Nodes[i].Layer) < layerOrder(response.Nodes[j].Layer)
		}
		if response.Nodes[i].Path != response.Nodes[j].Path {
			return response.Nodes[i].Path < response.Nodes[j].Path
		}
		return response.Nodes[i].Line < response.Nodes[j].Line
	})
	sort.Slice(response.Edges, func(i, j int) bool {
		if response.Edges[i].From != response.Edges[j].From {
			return response.Edges[i].From < response.Edges[j].From
		}
		if response.Edges[i].To != response.Edges[j].To {
			return response.Edges[i].To < response.Edges[j].To
		}
		return response.Edges[i].Kind < response.Edges[j].Kind
	})
	return response
}

func layerOrder(layer string) int {
	switch layer {
	case "ui":
		return 0
	case "api":
		return 1
	case "service":
		return 2
	case "db":
		return 3
	default:
		return 4
	}
}

func (s *Server) handleFlow(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "working"
	}
	if scope == "thread" && !s.threadsOrFail(w) {
		return
	}
	sources, id, status, message := s.flowSources(r, scope)
	if status != 0 {
		fail(w, status, message)
		return
	}
	writeJSON(w, s.buildFlow(r.Context(), sources, scope, id))
}

// Package route extracts framework routing (web frameworks: route -> handler
// references; routers: function -> screen navigates) into the 005 symbols/
// edges graph. It is additive on top of the 005 extractors: it never rewrites
// a 005 symbol or edge, and it applies a crisp three-way Confidence policy to
// references:
//
//	EXTRACTED  — explicit syntax / a same-file handler (a direct reference).
//	INFERRED   — a convention/heuristic (e.g. a markup-written navigates link).
//	AMBIGUOUS  — a name-only best-effort guess that resolved to a unique global
//	             symbol (stored, but low-confidence).
//
// A reference that is UNRESOLVABLE (no same-file match, no explicit
// specifier, and no unique global/owner-qualified match) is dropped at
// extraction and never stored, so a false positive can never inflate blast
// radius (drop-not-guess).
package route

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Edge kinds emitted by the route extractor (distinct from 005's call/heritage
// kinds).
const (
	KindReferences = "references"
	KindNavigates  = "navigates"
)

// Confidence labels (mirrored from extract/graph).
//
//	EXTRACTED  = explicit syntax (same-file handler, programmatic literal).
//	INFERRED   = convention/heuristic (markup-written navigates link).
//	AMBIGUOUS  = name-only best-effort guess (a unique global symbol was taken
//	            as the handler without an explicit specifier — low confidence).
//	Dropped    = unresolvable (not stored; not a label).
const (
	ConfidenceExtracted = "EXTRACTED"
	ConfidenceInferred  = "INFERRED"
	ConfidenceAmbiguous = "AMBIGUOUS"
)

// Symbol kinds assigned by the route extractor.
const (
	KindRoute  = "route"
	ScreenKind = "screen"
)

// Route is one URL -> handler binding for a web framework.
type Route struct {
	Framework   string
	Method      string
	PathPattern string
	Handler     string
	Line        int
	// Explicit is true when the route names the handler as a direct reference
	// (Django's as_view, FastAPI's function decorator). When false the handler
	// was bound by a framework convention (e.g. Rails to:) and is resolved with
	// a lower confidence.
	Explicit bool
}

// Navigation is one screen -> screen link produced by a frontend router.
type Navigation struct {
	Framework string
	Screen    string
	Dest      string
	// Markup is true when the link is written in markup (<Link>, <Route>)
	// rather than a programmatic call. Markup links are convention-derived
	// (INFERRED); programmatic literals are EXTRACTED.
	Markup bool
	Line   int
}

// FileRoutes is the extraction result for one file.
type FileRoutes struct {
	Framework string
	Routes    []Route
	Navigates []Navigation
}

// Extractor recognizes routing patterns in one file and returns its route
// nodes / navigates edges. Implementations must never error: a malformed file
// yields an empty (non-nil) FileRoutes so the index continues.
type Extractor interface {
	Extract(path string, src []byte) *FileRoutes
}

// ExtractFile dispatches path to the first framework handler whose file-type
// predicate matches. An unmatched or unsupported file yields an empty result.
// It never returns nil and never errors — a malformed file falls back to an
// empty result (the index continues).
func ExtractFile(path string, src []byte) *FileRoutes {
	out := &FileRoutes{}
	for _, h := range handlers {
		if !h.isFile(path) {
			continue
		}
		if src == nil {
			out.Framework = h.name
			return out
		}
		fr := h.extract(path, src)
		if fr == nil {
			out.Framework = h.name
			return out
		}
		fr.Framework = h.name
		out.Routes = fr.Routes
		out.Navigates = fr.Navigates
		out.Framework = h.name
		return out
	}
	return out
}

// handler is one framework's file-type predicate + route/navigates extractor.
type handler struct {
	name    string
	isFile  func(path string) bool
	extract func(path string, src []byte) *FileRoutes
}

// handlers is the per-framework dispatch table, ordered so more specific file
// types win (SvelteKit .svelte before the JS routers; Next.js before React
// Router for .tsx/.jsx).
var handlers = []handler{
	{"django", isPython, extractDjango},
	{"express", isJS, extractExpress},
	{"gin", isGo, extractGin},
	{"nethttp", isGo, extractNetHTTP},
	{"rails", isRuby, extractRails},
	{"spring", isJava, extractSpring},
	{"sveltekit", isSvelte, extractSvelteKit},
	{"nextjs", isJSX, extractNextJS},
	{"react-router", isJSX, extractReactRouter},
	{"vue-router", isJS, extractVueRouter},
}

func isPython(path string) bool { return filepath.Ext(path) == ".py" }

func isJS(path string) bool {
	switch filepath.Ext(path) {
	case ".js", ".mjs", ".cjs":
		return true
	}
	return false
}
func isJSX(path string) bool    { return filepath.Ext(path) == ".tsx" || filepath.Ext(path) == ".jsx" }
func isGo(path string) bool     { return filepath.Ext(path) == ".go" }
func isRuby(path string) bool   { return filepath.Ext(path) == ".rb" }
func isJava(path string) bool   { return filepath.Ext(path) == ".java" }
func isSvelte(path string) bool { return filepath.Ext(path) == ".svelte" }

// symbolUID derives a stable, path-independent identity for a route node from
// its framework + pattern + method (content excluded, so a moved file keeps
// its route identity).
func symbolUID(framework, pattern, method string) string {
	h := fnv.New64a()
	h.Write([]byte(framework + "\x00" + pattern + "\x00" + method))
	return fmt.Sprintf("%x", h.Sum64())
}

// screenPath derives a screen's stable route path from a named destination. A
// literal path (/about) is used as-is; a name-only destination ("settings") is
// normalized to a path-like screen key. This is a convention, not a claim of
// syntax — the destination is never fabricated beyond what the source holds.
func screenPath(dest string) string {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return ""
	}
	if strings.HasPrefix(dest, "/") {
		return dest
	}
	return "/" + dest
}

// lineOf returns the 1-based line number of byte offset off in src.
func lineOf(src []byte, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	n := 1
	for i := 0; i < off; i++ {
		if src[i] == '\n' {
			n++
		}
	}
	return n
}

// absPattern normalizes a relative URL pattern (Django) to an absolute one.
// Django's urls.py patterns are conventionally relative ("home/"); the route
// node key is the absolute form ("/home/") so a query for the URL pattern is
// unambiguous.
func absPattern(p string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}

// stripQuotes removes a single layer of surrounding matching quotes.
func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

var idRe = regexp.MustCompile(`^[A-Za-z_]\w*$`)

// --- Web framework: route -> handler references ---

var djangoURLsRe = regexp.MustCompile(`(?m)^\s*(?:re\.path|path)\s*\(\s*['\"]([^'\"]*)['\"]\s*,\s*([^)]+?)\s*\)\s*,?\s*$`)
var djangoAsViewRe = regexp.MustCompile(`([\w.]+)\.as_view\(\s*\)`)
var djangoNameRe = regexp.MustCompile(`\bname\s*=\s*['\"]([^'\"]*)['\"]`)
var djangoAsViewNameRe = regexp.MustCompile(`(?m)^\s*(?:re\.path|path)\s*\(\s*['\"]([^'\"]*)['\"]\s*,\s*([\w.]+)\.as_view\(\s*\)\s*(?:,[^)\n]*)?\)\s*,?\s*$`)

func extractDjango(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "django"}
	text := string(src)
	// Direct as_view() forms (single-line entries) — explicit references.
	for _, m := range djangoAsViewNameRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 4 {
			continue
		}
		pattern := absPattern(text[m[2]:m[3]])
		handler := strings.TrimPrefix(text[m[4]:m[5]], "views.")
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "django", Method: "GET", PathPattern: pattern,
			Handler: handler, Line: lineOf(src, m[0]), Explicit: true,
		})
	}
	// Generic entries (view references / computed) for the non-as_view forms.
	for _, m := range djangoURLsRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 4 {
			continue
		}
		pattern := absPattern(text[m[2]:m[3]])
		second := text[m[4]:m[5]]
		if djangoAsViewRe.MatchString(second) {
			continue // already handled by the explicit pass
		}
		handler := djangoHandler(second)
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "django", Method: "GET", PathPattern: pattern,
			Handler: handler, Line: lineOf(src, m[0]), Explicit: true,
		})
	}
	// FastAPI decorators in the same .py (no URL entries present) are also
	// valid route sources — union them in so a FastAPI module is not silently
	// dropped by the Django dispatch.
	if len(fr.Routes) == 0 {
		fr.Routes = fastAPIRoutes(src)
	}
	sortRoutes(fr)
	return fr
}

// fastAPIRoutes returns the routes for a FastAPI module (@app.get decorators).
// It is shared by the python dispatch so a .py file that is FastAPI (no
// urlpatterns) still yields its routes.
func fastAPIRoutes(src []byte) []Route {
	var routes []Route
	lines := strings.Split(string(src), "\n")
	for i := 0; i < len(lines); i++ {
		m := fastapiDecoratorRe.FindStringSubmatchIndex(lines[i])
		if m == nil {
			continue
		}
		method := strings.ToUpper(lines[i][m[2]:m[3]])
		pattern := lines[i][m[4]:m[5]]
		handler := nextFuncName(lines, i+1)
		if handler == "" {
			continue
		}
		routes = append(routes, Route{
			Framework: "fastapi", Method: method, PathPattern: pattern,
			Handler: handler, Line: i + 1, Explicit: true,
		})
	}
	return routes
}

// djangoHandler returns the handler symbol name from a URL entry's second
// argument: a dotted view reference or a bare identifier. A computed argument
// (neither shape) yields "" (the reference is then dropped).
func djangoHandler(second string) string {
	second = djangoNameRe.ReplaceAllString(second, "")
	second = strings.TrimSpace(second)
	second = strings.TrimPrefix(second, "views.")
	if idRe.MatchString(second) {
		return second
	}
	return ""
}

var fastapiDecoratorRe = regexp.MustCompile(`@app\.(get|post|put|delete|patch)\(\s*['\"]([^'\"]*)['\"]`)
var funcDefRe = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(`)

func nextFuncName(lines []string, idx int) string {
	for i := idx; i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if l == "" {
			continue
		}
		if m := funcDefRe.FindStringSubmatch(l); m != nil {
			return m[1]
		}
		return ""
	}
	return ""
}

var expressRouteRe = regexp.MustCompile(`(?m)^\s*app\.(get|post|put|delete|patch)\s*\(\s*['\"]([^'\"]*)['\"]\s*,\s*([^)]+?)\s*\)\s*[,;]?\s*$`)

func extractExpress(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "express"}
	for _, m := range expressRouteRe.FindAllStringSubmatchIndex(string(src), -1) {
		if m == nil || len(m) < 6 {
			continue
		}
		method := strings.ToUpper(string(src[m[2]:m[3]]))
		pattern := string(src[m[4]:m[5]])
		handler := routeHandler(string(src[m[6]:m[7]]))
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "express", Method: method, PathPattern: pattern,
			Handler: handler, Line: lineOf(src, m[0]), Explicit: true,
		})
	}
	// A .js file that is Vue Router (router.push) but not Express (no
	// app.get) still yields its navigations — union them in so the first
	// .js handler does not silently drop the others.
	if len(fr.Routes) == 0 && len(fr.Navigates) == 0 {
		fr.Navigates = vueRouterNavigates(src)
	}
	return fr
}

// routeHandler returns the handler function name from a route's handler
// argument: a bare identifier, or the first named entry of a middleware array.
// A computed argument (arrow function, etc.) yields "".
func routeHandler(arg string) string {
	arg = strings.TrimSpace(arg)
	if strings.HasPrefix(arg, "[") {
		inner := strings.TrimSuffix(strings.TrimPrefix(arg, "["), "]")
		for _, part := range strings.Split(inner, ",") {
			part = strings.TrimSpace(part)
			if idRe.MatchString(part) {
				return part
			}
		}
		return ""
	}
	if idRe.MatchString(arg) {
		return arg
	}
	return ""
}

var vuePushNameRe = regexp.MustCompile(`\brouter\.(?:push|replace)\(\s*\{[^}]*\bname\s*:\s*['\"]([^'\"]*)['\"]`)
var vuePushPathRe = regexp.MustCompile(`\brouter\.(?:push|replace)\(\s*['\"]([^'\"]*)['\"]`)

// vueRouterNavigates returns the navigations for a Vue Router module
// (router.push({name}) / router.push('/x')).
func vueRouterNavigates(src []byte) []Navigation {
	var navs []Navigation
	text := string(src)
	for _, m := range vuePushNameRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		navs = append(navs, Navigation{
			Framework: "vue-router", Screen: screenPath(dest), Dest: dest,
			Line: lineOf(src, m[0]),
		})
	}
	for _, m := range vuePushPathRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		navs = append(navs, Navigation{
			Framework: "vue-router", Screen: screenPath(dest), Dest: dest,
			Line: lineOf(src, m[0]),
		})
	}
	return navs
}

var ginRouteRe = regexp.MustCompile(`(?m)^\s*\b[rw]\.(GET|POST|PUT|DELETE|PATCH|Any|HEAD|OPTIONS)\s*\(\s*["\x27]([^"\x27]*)["\x27]\s*,\s*([^)]+?)\s*\)\s*[,;]?\s*$`)

func extractGin(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "gin"}
	for _, m := range ginRouteRe.FindAllStringSubmatchIndex(string(src), -1) {
		if m == nil || len(m) < 6 {
			continue
		}
		method := strings.ToUpper(string(src[m[2]:m[3]]))
		pattern := string(src[m[4]:m[5]])
		handler := routeHandler(string(src[m[6]:m[7]]))
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "gin", Method: method, PathPattern: pattern,
			Handler: handler, Line: lineOf(src, m[0]), Explicit: true,
		})
	}
	// A .go file with no gin binding (r.GET/...) may be a stdlib nethttp
	// server instead — union the nethttp routes in so the gin dispatch does
	// not silently drop them. The route-level Framework stays "nethttp".
	if len(fr.Routes) == 0 && len(fr.Navigates) == 0 {
		fr.Routes = append(fr.Routes, extractNetHTTP(path, src).Routes...)
	}
	return fr
}

// nethttpRouteRe matches stdlib mux bindings: `mux.HandleFunc("GET /path",
// s.handler)` (Go 1.22 "METHOD /path") and `mux.HandleFunc("/path", handler)`
// (Go 1.21, method defaults to ANY). The handler arg is captured non-greedily
// up to the first `)`, so a wrapping call like `s.requireAuth(s.handleX)`
// yields `s.requireAuth(s.handleX` — nethttpHandler takes the last identifier
// in it, which is the inner handler.
var nethttpRouteRe = regexp.MustCompile(`(?m)^\s*[\w.]*\.?(HandleFunc|Handle)\s*\(\s*["\x27]([A-Z]+\s+)?([^"\x27]+)["\x27]\s*,\s*([^)]+?)\s*\)`)

func extractNetHTTP(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "nethttp"}
	for _, m := range nethttpRouteRe.FindAllStringSubmatchIndex(string(src), -1) {
		if m == nil || len(m) < 8 {
			continue
		}
		method := "ANY"
		if m[4] >= 0 {
			method = strings.ToUpper(strings.TrimSpace(string(src[m[4]:m[5]])))
		}
		pattern := string(src[m[6]:m[7]])
		handler := nethttpHandler(string(src[m[8]:m[9]]))
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "nethttp", Method: method, PathPattern: pattern,
			Handler: handler, Line: lineOf(src, m[0]), Explicit: true,
		})
	}
	return fr
}

// nethttpHandler returns the handler symbol name from a HandleFunc/Handle
// argument: the LAST identifier run in the arg (a bare identifier, the member
// of a selector like `s.handleX`, or the innermost call of a wrapping
// expression like `s.requireAuth(s.handleX`). A computed arg (lambda, call
// with no identifier) yields "".
func nethttpHandler(arg string) string {
	var last string
	for _, id := range idRuns.FindAllString(arg, -1) {
		if goKeyword[id] {
			continue
		}
		last = id
	}
	return last
}

var idRuns = regexp.MustCompile(`[A-Za-z_]\w*`)

var goKeyword = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true,
	"continue": true, "default": true, "defer": true, "else": true,
	"fallthrough": true, "for": true, "func": true, "go": true,
	"goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true,
	"var": true,
}

var railsRouteRe = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|resources|resource)\s+['\"]([^'\"]*)['\"]`)
var railsToRe = regexp.MustCompile(`\bto\s*:\s*['\"]([^'\"]+)['\"]`)

func extractRails(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "rails"}
	text := string(src)
	for _, m := range railsRouteRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 4 {
			continue
		}
		method := strings.ToUpper(text[m[2]:m[3]])
		pattern := text[m[4]:m[5]]
		rest := text[m[1]:]
		explicit := false
		handler := ""
		if tm := railsToRe.FindStringSubmatch(rest); tm != nil {
			handler = railsTarget(tm[1])
			explicit = true
		} else {
			handler = railsConvention(method, pattern)
		}
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "rails", Method: method, PathPattern: pattern,
			Handler: handler, Line: lineOf(src, m[0]), Explicit: explicit,
		})
	}
	return fr
}

// railsTarget returns a stable handler symbol name for a `to:` target:
// "Controller#Action" becomes "Controller#Action" (the full action ref, which
// is the natural symbol identity for the handler) and a bare "action" is kept
// as-is.
func railsTarget(target string) string {
	return strings.TrimSpace(target)
}

func railsConvention(method, pattern string) string {
	p := strings.TrimLeft(pattern, "/")
	if p == "" {
		return "root#index"
	}
	return p + "#index"
}

var springMappingRe = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Request)Mapping`)
var springPathRe = regexp.MustCompile(`@(?:Get|Post|Put|Delete|Patch|Request)Mapping\s*\(\s*(?:value\s*=\s*)?["\x27]([^"\x27]*)["\x27]`)
var javaMethodRe = regexp.MustCompile(`^(?:public\s+|private\s+|protected\s+)?[\w<>\[\],\s]+\s([A-Za-z_]\w*)\s*\([^)]*\)\s*\{`)

func extractSpring(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "spring"}
	lines := strings.Split(string(src), "\n")
	for i := 0; i < len(lines); i++ {
		if !springMappingRe.MatchString(lines[i]) {
			continue
		}
		pm := springPathRe.FindStringSubmatch(lines[i])
		if pm == nil {
			continue
		}
		method := ""
		if mm := springMappingRe.FindStringSubmatch(lines[i]); mm != nil {
			method = strings.ToUpper(mm[1])
		}
		handler := nextJavaMethod(lines, i+1)
		if handler == "" {
			continue
		}
		fr.Routes = append(fr.Routes, Route{
			Framework: "spring", Method: method, PathPattern: pm[1],
			Handler: handler, Line: i + 1, Explicit: true,
		})
	}
	return fr
}

func nextJavaMethod(lines []string, idx int) string {
	for i := idx; i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if l == "" {
			continue
		}
		if m := javaMethodRe.FindStringSubmatch(l); m != nil {
			return m[1]
		}
		return ""
	}
	return ""
}

// --- Routers: function -> screen navigates ---

var nextPushRe = regexp.MustCompile(`router\.push\(\s*['\"]([^'\"]*)['\"]`)
var linkHrefRe = regexp.MustCompile(`<Link[^>]*\bhref\s*=\s*['\"]([^'\"]*)['\"]`)

func extractNextJS(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "nextjs"}
	text := string(src)
	for _, m := range nextPushRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		fr.Navigates = append(fr.Navigates, Navigation{
			Framework: "nextjs", Screen: screenPath(dest), Dest: dest,
			Line: lineOf(src, m[0]),
		})
	}
	for _, m := range linkHrefRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		fr.Navigates = append(fr.Navigates, Navigation{
			Framework: "nextjs", Screen: screenPath(dest), Dest: dest,
			Markup: true, Line: lineOf(src, m[0]),
		})
	}
	// A .tsx file that is React Router (useNavigate / <Route>) but not Next
	// (no router.push / <Link>) still yields its navigations — union them in
	// so the first .tsx/.jsx handler does not silently drop the others.
	if len(fr.Navigates) == 0 {
		fr.Navigates = reactRouterNavigates(src)
	}
	return fr
}

// reactRouterNavigates returns the navigations for a React Router module
// (<Route path> definitions + navigate('/x') calls).
func reactRouterNavigates(src []byte) []Navigation {
	var navs []Navigation
	text := string(src)
	for _, m := range routePathRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		navs = append(navs, Navigation{
			Framework: "react-router", Screen: screenPath(dest), Dest: dest,
			Markup: true, Line: lineOf(src, m[0]),
		})
	}
	for _, m := range reactNavRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		navs = append(navs, Navigation{
			Framework: "react-router", Screen: screenPath(dest), Dest: dest,
			Line: lineOf(src, m[0]),
		})
	}
	return navs
}

var routePathRe = regexp.MustCompile(`<Route[^>]*\bpath\s*=\s*['\"]([^'\"]*)['\"]`)
var reactNavRe = regexp.MustCompile(`\bnavigate\(\s*['\"]([^'\"]*)['\"]`)

func extractReactRouter(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "react-router"}
	fr.Navigates = reactRouterNavigates(src)
	return fr
}

var svelteGotoRe = regexp.MustCompile(`goto\(\s*['\"]([^'\"]*)['\"]`)

func extractSvelteKit(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "sveltekit"}
	text := string(src)
	for _, m := range svelteGotoRe.FindAllStringSubmatchIndex(text, -1) {
		if m == nil || len(m) < 3 {
			continue
		}
		dest := text[m[2]:m[3]]
		fr.Navigates = append(fr.Navigates, Navigation{
			Framework: "sveltekit", Screen: screenPath(dest), Dest: dest,
			Line: lineOf(src, m[0]),
		})
	}
	return fr
}

func extractVueRouter(path string, src []byte) *FileRoutes {
	fr := &FileRoutes{Framework: "vue-router"}
	fr.Navigates = vueRouterNavigates(src)
	return fr
}

// sortRoutes makes route/navigation output deterministic for a given file.
func sortRoutes(fr *FileRoutes) {
	sort.SliceStable(fr.Routes, func(i, j int) bool {
		if fr.Routes[i].Line != fr.Routes[j].Line {
			return fr.Routes[i].Line < fr.Routes[j].Line
		}
		return fr.Routes[i].PathPattern < fr.Routes[j].PathPattern
	})
	sort.SliceStable(fr.Navigates, func(i, j int) bool {
		if fr.Navigates[i].Line != fr.Navigates[j].Line {
			return fr.Navigates[i].Line < fr.Navigates[j].Line
		}
		return fr.Navigates[i].Dest < fr.Navigates[j].Dest
	})
}

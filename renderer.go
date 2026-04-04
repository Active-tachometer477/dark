package dark

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/i2y/ramune"
)

// metafileOutput represents a single output entry in esbuild's metafile JSON.
type metafileOutput struct {
	EntryPoint string `json:"entryPoint"`
}

// metafileJSON represents the top-level structure of esbuild's metafile.
type metafileJSON struct {
	Outputs map[string]metafileOutput `json:"outputs"`
}

const requireShim = `globalThis.require = function(name) {
  var s = name.replace(/^@/, '').replace(/[\/-]/g, '_');
  var m = globalThis[s];
  if (m !== undefined) return m;
  throw new Error('dark: module not found: ' + name);
};`

// Global JS variable names for broadcast components.
const (
	globalLayout            = "__dark_layout"
	globalErrorComponent    = "__dark_error_component"
	globalNotFoundComponent = "__dark_not_found_component"
	layoutResolveJS         = "var __Layout = globalThis." + globalLayout + ".default || globalThis." + globalLayout + ";\n"
)

type layoutEntry struct {
	globalName string // e.g. "__dark_rlayout_abc123"
	css        string
}

type renderer struct {
	pool                 *ramune.RuntimePool
	uikit                *uikit
	templateDir          string
	devMode              bool
	hasLayout            bool
	hasIslands           bool
	hasErrorComponent    bool
	hasNotFoundComponent bool
	clientChunks         map[string]string // filename → JS content
	clientManifest       map[string]string // island name → chunk filename
	clientEntryFiles     map[string]bool   // set of entry filenames (vs shared chunks)
	clientChunksMu       sync.RWMutex
	nodeModulesDir       string

	layoutCSS   string
	layoutCSSMu sync.RWMutex
	errorCSS    string
	notFoundCSS string

	routeLayoutsMu sync.RWMutex
	routeLayouts   map[string]*layoutEntry // absolute path → entry

	pageCSSMu sync.RWMutex
	pageCSS   map[string]string // content-hash → CSS content

	cacheMu sync.RWMutex
	cache   map[string]*cacheEntry

	ssrCache *lruCache[ssrCacheEntry]
}

type ssrCacheEntry struct {
	html string
	css  string
}

type cacheEntry struct {
	bundledJS  string
	bundledCSS string
	modTime    int64
	srcMap     *sourceMap // parsed source map (dev mode only)
}

func newRenderer(cfg *config) (*renderer, error) {
	kit := resolveUIKit(cfg.uiLibrary)
	deps := append([]string{}, kit.ssrDeps...)
	deps = append(deps, cfg.extraDeps...)

	poolOpts := []ramune.Option{
		ramune.Dependencies(deps...),
		ramune.WithPermissions(ramune.SandboxPermissions()),
	}
	if kit.preloadJS != "" {
		poolOpts = append(poolOpts, ramune.PreloadJS(kit.preloadJS))
	}

	pool, err := ramune.NewPool(cfg.poolSize, poolOpts...)
	if err != nil {
		return nil, fmt.Errorf("dark: failed to create runtime pool: %w", err)
	}

	r := &renderer{
		pool:         pool,
		uikit:        kit,
		templateDir:  cfg.templateDir,
		devMode:      cfg.devMode,
		cache:        make(map[string]*cacheEntry),
		pageCSS:      make(map[string]string),
		routeLayouts: make(map[string]*layoutEntry),
		ssrCache:     newLRUCache[ssrCacheEntry](cfg.ssrCacheSize),
	}

	// Broadcast require shim to all workers.
	if err := pool.Broadcast(requireShim); err != nil {
		pool.Close()
		return nil, fmt.Errorf("dark: failed to broadcast require shim: %w", err)
	}

	// Bundle and broadcast global components.
	if cfg.layoutFile != "" {
		css, err := r.bundleAndBroadcast(cfg.layoutFile, globalLayout)
		if err != nil {
			pool.Close()
			return nil, err
		}
		r.hasLayout = true
		r.layoutCSS = css
	}
	if cfg.errorComponent != "" {
		css, err := r.bundleAndBroadcast(cfg.errorComponent, globalErrorComponent)
		if err != nil {
			pool.Close()
			return nil, err
		}
		r.hasErrorComponent = true
		r.errorCSS = css
	}
	if cfg.notFoundComponent != "" {
		css, err := r.bundleAndBroadcast(cfg.notFoundComponent, globalNotFoundComponent)
		if err != nil {
			pool.Close()
			return nil, err
		}
		r.hasNotFoundComponent = true
		r.notFoundCSS = css
	}

	return r, nil
}

// bundleAndBroadcast bundles a TSX file and broadcasts it as a global to all workers.
func (r *renderer) bundleAndBroadcast(relPath, globalName string) (css string, err error) {
	fullPath := filepath.Join(r.templateDir, relPath)
	js, css, err := r.bundleComponent(fullPath)
	if err != nil {
		return "", fmt.Errorf("dark: failed to bundle %s: %w", relPath, err)
	}
	code := strings.Replace(js, "var __dark_mod", "globalThis."+globalName, 1)
	if err := r.pool.Broadcast(code); err != nil {
		return "", fmt.Errorf("dark: failed to broadcast %s: %w", relPath, err)
	}
	return css, nil
}

// prepareRouteLayouts bundles and broadcasts all unique route-specific layouts.
func (r *renderer) prepareRouteLayouts(routes []registeredRoute) error {
	seen := make(map[string]bool)
	for _, rt := range routes {
		if rt.page == nil || rt.page.Layout == "" {
			continue
		}
		// Layout may be comma-separated (from Group nesting).
		for _, layout := range strings.Split(rt.page.Layout, ",") {
			layout = strings.TrimSpace(layout)
			if layout == "" || seen[layout] {
				continue
			}
			seen[layout] = true

			absPath, err := filepath.Abs(filepath.Join(r.templateDir, layout))
			if err != nil {
				return fmt.Errorf("dark: failed to resolve route layout path %s: %w", layout, err)
			}

			js, css, err := r.bundleComponent(absPath)
			if err != nil {
				return fmt.Errorf("dark: failed to bundle route layout %s: %w", layout, err)
			}

			// Create a deterministic global name from hash of the layout path.
			h := sha256.Sum256([]byte(absPath))
			globalName := "__dark_rlayout_" + hex.EncodeToString(h[:])[:12]

			code := strings.Replace(js, "var __dark_mod", "globalThis."+globalName, 1)
			if err := r.pool.Broadcast(code); err != nil {
				return fmt.Errorf("dark: failed to broadcast route layout %s: %w", layout, err)
			}

			r.routeLayoutsMu.Lock()
			r.routeLayouts[absPath] = &layoutEntry{globalName: globalName, css: css}
			r.routeLayoutsMu.Unlock()
		}
	}
	return nil
}

// reloadRouteLayout re-bundles and re-broadcasts a single route layout.
func (r *renderer) reloadRouteLayout(layoutFile string) error {
	absPath, err := filepath.Abs(filepath.Join(r.templateDir, layoutFile))
	if err != nil {
		return err
	}

	r.routeLayoutsMu.RLock()
	entry, ok := r.routeLayouts[absPath]
	r.routeLayoutsMu.RUnlock()
	if !ok {
		return nil // not a registered route layout
	}

	js, css, err := r.bundleComponent(absPath)
	if err != nil {
		return fmt.Errorf("dark: failed to rebundle route layout %s: %w", layoutFile, err)
	}

	code := strings.Replace(js, "var __dark_mod", "globalThis."+entry.globalName, 1)
	if err := r.pool.Broadcast(code); err != nil {
		return fmt.Errorf("dark: failed to re-broadcast route layout %s: %w", layoutFile, err)
	}

	r.routeLayoutsMu.Lock()
	entry.css = css
	r.routeLayoutsMu.Unlock()

	r.invalidateAll()
	return nil
}

// isRouteLayout checks if an absolute path is a registered route layout.
func (r *renderer) isRouteLayout(absPath string) bool {
	r.routeLayoutsMu.RLock()
	defer r.routeLayoutsMu.RUnlock()
	_, ok := r.routeLayouts[absPath]
	return ok
}

func (r *renderer) close() error {
	return r.pool.Close()
}

func (r *renderer) render(componentPath string, routeLayouts []string, props any, skipLayout bool) (string, string, error) {
	fullPath := filepath.Join(r.templateDir, componentPath)

	bundledJS, bundledCSS, err := r.getCachedBundle(fullPath)
	if err != nil {
		return "", "", err
	}

	propsJSON, err := json.Marshal(props)
	if err != nil {
		return "", "", fmt.Errorf("dark: failed to marshal props: %w", err)
	}

	// SSR output cache: check before expensive pool.Eval().
	var ssrKey string
	if r.ssrCache.maxSize > 0 {
		ssrKey = r.ssrCacheKey(componentPath, routeLayouts, skipLayout, propsJSON)
		if entry, ok := r.ssrCache.get(ssrKey); ok {
			return entry.html, entry.css, nil
		}
	}

	var code strings.Builder
	fmt.Fprintf(&code, "var __props = %s;\n", propsJSON)
	fmt.Fprintf(&code, "%s\n", bundledJS)
	code.WriteString(r.uikit.rtsResolveJS)
	code.WriteString("var __Component = __dark_mod.default || __dark_mod;\n")

	// Build the innermost expression first: createElement(__Component, __props)
	// Then wrap with route layouts (innermost first), then global layout (outermost).
	ce := r.uikit.createElement
	inner := ce + "(__Component, __props)"

	if !skipLayout {
		// Wrap with route layouts (innermost first, iterate in reverse).
		for i := len(routeLayouts) - 1; i >= 0; i-- {
			_, entry, ok := r.resolveRouteLayout(routeLayouts[i])
			if !ok {
				return "", "", fmt.Errorf("dark: route layout not prepared: %s", routeLayouts[i])
			}
			varName := fmt.Sprintf("__RL%d", i)
			fmt.Fprintf(&code, "var %s = globalThis.%s.default || globalThis.%s;\n", varName, entry.globalName, entry.globalName)
			inner = fmt.Sprintf("%s(%s, Object.assign({}, __props, { children: %s }))", ce, varName, inner)
		}

		if r.hasLayout {
			code.WriteString(layoutResolveJS)
			inner = fmt.Sprintf("%s(__Layout, Object.assign({}, __props, { children: %s }))", ce, inner)
		}
	}

	fmt.Fprintf(&code, "__rts(%s);\n", inner)

	val, err := r.pool.Eval(code.String())
	if err != nil {
		if r.devMode {
			if sm := r.getSourceMap(componentPath); sm != nil {
				mapped := mapErrorWithSourceMap(err.Error(), sm)
				return "", "", fmt.Errorf("dark: render %s: %s\n%w", componentPath, mapped, err)
			}
		}
		return "", "", fmt.Errorf("dark: render %s: %w", componentPath, err)
	}
	defer val.Close()

	html, err := val.GoString()
	if err != nil {
		return "", "", fmt.Errorf("dark: render %s: failed to get HTML string: %w", componentPath, err)
	}

	// Inject hydration script if this page uses islands.
	if r.hasIslands && !skipLayout && strings.Contains(html, "<dark-island") {
		html = r.injectIslandsScript(html)
	}

	// Collect CSS: layout CSS + component CSS.
	var css string
	if !skipLayout {
		css = r.collectLayoutCSS(routeLayouts)
	}
	if bundledCSS != "" {
		css += bundledCSS
	}

	// Store in SSR cache.
	if ssrKey != "" {
		r.putSSRCache(ssrKey, html, css)
	}

	return html, css, nil
}

// renderShell renders the layout(s) with a placeholder marker instead of children.
// Returns the HTML split into before/after the marker, plus combined layout CSS.
func (r *renderer) renderShell(routeLayouts []string, props any) (before string, after string, shellCSS string, err error) {
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return "", "", "", fmt.Errorf("dark: failed to marshal props: %w", err)
	}

	var code strings.Builder
	fmt.Fprintf(&code, "var __props = %s;\n", propsJSON)
	code.WriteString(r.uikit.rtsResolveJS)

	// Build nested layout expression with a marker element as the innermost child.
	ce := r.uikit.createElement
	inner := ce + "('dark-stream-marker', null)"

	// Wrap with route layouts (innermost first, iterate in reverse).
	for i := len(routeLayouts) - 1; i >= 0; i-- {
		_, entry, ok := r.resolveRouteLayout(routeLayouts[i])
		if !ok {
			return "", "", "", fmt.Errorf("dark: route layout not prepared: %s", routeLayouts[i])
		}
		varName := fmt.Sprintf("__RL%d", i)
		fmt.Fprintf(&code, "var %s = globalThis.%s.default || globalThis.%s;\n", varName, entry.globalName, entry.globalName)
		inner = fmt.Sprintf("%s(%s, Object.assign({}, __props, { children: %s }))", ce, varName, inner)
	}

	if r.hasLayout {
		code.WriteString(layoutResolveJS)
		inner = fmt.Sprintf("%s(__Layout, Object.assign({}, __props, { children: %s }))", ce, inner)
	}

	fmt.Fprintf(&code, "__rts(%s);\n", inner)

	val, err := r.pool.Eval(code.String())
	if err != nil {
		return "", "", "", fmt.Errorf("dark: renderShell: %w", err)
	}
	defer val.Close()

	html, err := val.GoString()
	if err != nil {
		return "", "", "", fmt.Errorf("dark: renderShell: failed to get HTML string: %w", err)
	}

	// Split at the marker.
	const marker = "<dark-stream-marker></dark-stream-marker>"
	idx := strings.Index(html, marker)
	if idx < 0 {
		return html, "", "", nil
	}

	before = html[:idx]
	after = html[idx+len(marker):]

	return before, after, r.collectLayoutCSS(routeLayouts), nil
}

func (r *renderer) getCachedBundle(fullPath string) (string, string, error) {
	r.cacheMu.RLock()
	entry, ok := r.cache[fullPath]
	r.cacheMu.RUnlock()

	if ok {
		if !r.devMode {
			return entry.bundledJS, entry.bundledCSS, nil
		}
		info, err := os.Stat(fullPath)
		if err == nil && info.ModTime().UnixNano() == entry.modTime {
			return entry.bundledJS, entry.bundledCSS, nil
		}
	}

	bundledJS, bundledCSS, err := r.bundleComponent(fullPath)
	if err != nil {
		return "", "", err
	}

	var modTime int64
	if info, err := os.Stat(fullPath); err == nil {
		modTime = info.ModTime().UnixNano()
	}

	if r.devMode {
		sm, _ := parseInlineSourceMap(bundledJS)
		if sm != nil {
			bundledJS = stripInlineSourceMap(bundledJS)
		}
		ce := &cacheEntry{bundledJS: bundledJS, bundledCSS: bundledCSS, modTime: modTime, srcMap: sm}
		r.cacheMu.Lock()
		r.cache[fullPath] = ce
		r.cacheMu.Unlock()
		return bundledJS, bundledCSS, nil
	}

	r.cacheMu.Lock()
	r.cache[fullPath] = &cacheEntry{bundledJS: bundledJS, bundledCSS: bundledCSS, modTime: modTime}
	r.cacheMu.Unlock()

	return bundledJS, bundledCSS, nil
}

func (r *renderer) bundleComponent(filePath string) (string, string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", "", fmt.Errorf("dark: failed to resolve path %s: %w", filePath, err)
	}

	// Base externals for SSR: UI library packages are provided as globals.
	exactExternals := append([]string{}, r.uikit.ssrExternals...)
	if r.hasIslands {
		exactExternals = append(exactExternals, "dark")
	}

	opts := api.BuildOptions{
		EntryPoints: []string{absPath},
		Bundle:      true,
		Format:      api.FormatIIFE,
		GlobalName:  "__dark_mod",
		Platform:    api.PlatformBrowser,
		Write:       false,
		Outdir:      "/", // required for CSS extraction with Write:false
		Plugins:     []api.Plugin{exactExternalPlugin(exactExternals)},
		JSX:         api.JSXTransform,
		JSXFactory:  r.uikit.jsxFactory,
		JSXFragment: r.uikit.jsxFragment,
		LogLevel:    api.LogLevelSilent,
	}
	if r.devMode {
		opts.Sourcemap = api.SourceMapInline
	}

	// When islands are enabled, provide node_modules path so library subpaths
	// (e.g., preact/hooks) can be resolved and bundled inline (not externalized).
	if r.hasIslands && r.nodeModulesDir != "" {
		opts.NodePaths = []string{r.nodeModulesDir}
	}

	result := api.Build(opts)

	if len(result.Errors) > 0 {
		return "", "", fmt.Errorf("dark: esbuild error for %s: %s", filePath, result.Errors[0].Text)
	}
	if len(result.OutputFiles) == 0 {
		return "", "", fmt.Errorf("dark: esbuild produced no output for %s", filePath)
	}

	js, css := extractJSAndCSS(result.OutputFiles)
	return js, css, nil
}

// extractJSAndCSS splits esbuild output files into JS and CSS content.
func extractJSAndCSS(files []api.OutputFile) (js, css string) {
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".css") {
			css = strings.TrimSpace(string(f.Contents))
		} else {
			js = strings.TrimSpace(string(f.Contents))
		}
	}
	return js, css
}

// exactExternalPlugin creates an esbuild plugin that only externalizes exact
// package names, not their subpaths.
func exactExternalPlugin(pkgs []string) api.Plugin {
	set := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		set[p] = true
	}
	return api.Plugin{
		Name: "dark-exact-external",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if set[args.Path] {
					return api.OnResolveResult{Path: args.Path, External: true}, nil
				}
				return api.OnResolveResult{}, nil
			})
		},
	}
}

// buildIslands sets up island support.
func (r *renderer) buildIslands(islands []islandEntry, cfg *config) error {
	deps := append([]string{}, r.uikit.ssrDeps...)
	deps = append(deps, cfg.extraDeps...)
	cacheDir, err := islandCacheDir(deps)
	if err != nil {
		return fmt.Errorf("dark: failed to create island cache dir: %w", err)
	}

	nmDir := filepath.Join(cacheDir, "node_modules")
	if _, err := os.Stat(filepath.Join(nmDir, r.uikit.clientPkgCheck, "package.json")); err != nil {
		if err := ramune.InstallNpmPackages(r.uikit.clientPkg, cacheDir); err != nil {
			return fmt.Errorf("dark: failed to install client packages for islands: %w", err)
		}
	}

	r.nodeModulesDir = nmDir

	if err := r.pool.Broadcast(r.uikit.darkModuleJS); err != nil {
		return fmt.Errorf("dark: failed to broadcast dark module: %w", err)
	}

	r.hasIslands = true

	if err := r.buildClientBundle(islands, cfg, nmDir); err != nil {
		return fmt.Errorf("dark: failed to build client bundle: %w", err)
	}

	return nil
}

// buildClientBundle generates per-island ES module chunks using esbuild code splitting.
// Each island becomes a separate entry point; shared dependencies (Preact) are
// automatically extracted into shared chunks.
func (r *renderer) buildClientBundle(islands []islandEntry, cfg *config, nodeModulesDir string) error {
	absTemplateDir, err := filepath.Abs(cfg.templateDir)
	if err != nil {
		return fmt.Errorf("dark: failed to resolve template dir: %w", err)
	}

	// Create temp directory for per-island entry files.
	tmpDir, err := os.MkdirTemp("", "dark-islands-*")
	if err != nil {
		return fmt.Errorf("dark: failed to create temp dir for islands: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	entryPoints := make([]string, len(islands))
	for i, isl := range islands {
		entryCode := buildIslandEntryJS(isl, absTemplateDir, r.uikit, cfg.devMode)
		entryFile := filepath.Join(tmpDir, isl.name+".jsx")
		if err := os.WriteFile(entryFile, []byte(entryCode), 0o644); err != nil {
			return fmt.Errorf("dark: failed to write island entry %s: %w", isl.name, err)
		}
		entryPoints[i] = entryFile
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       entryPoints,
		Bundle:            true,
		Format:            api.FormatESModule,
		Splitting:         true,
		Platform:          api.PlatformBrowser,
		Write:             false,
		Outdir:            "/_dark/islands",
		EntryNames:        "[name]-[hash]",
		ChunkNames:        "chunk-[hash]",
		NodePaths:         []string{nodeModulesDir},
		AbsWorkingDir:     absTemplateDir,
		JSX:               api.JSXTransform,
		JSXFactory:        r.uikit.jsxFactory,
		JSXFragment:       r.uikit.jsxFragment,
		MinifySyntax:      !cfg.devMode,
		MinifyWhitespace:  !cfg.devMode,
		MinifyIdentifiers: !cfg.devMode,
		Metafile:          true,
		LogLevel:          api.LogLevelSilent,
	})

	if len(result.Errors) > 0 {
		return fmt.Errorf("dark: esbuild client bundle error: %s", result.Errors[0].Text)
	}
	if len(result.OutputFiles) == 0 {
		return fmt.Errorf("dark: esbuild produced no client bundle output")
	}

	// Build mapping from entry basename (without extension) to island name.
	entryBaseToName := make(map[string]string, len(islands))
	for _, isl := range islands {
		entryBaseToName[isl.name+".jsx"] = isl.name
	}

	// Parse metafile to build manifest.
	manifest, err := r.parseMetafile(result.Metafile, entryBaseToName)
	if err != nil {
		return fmt.Errorf("dark: failed to parse metafile: %w", err)
	}

	r.setClientChunks(result.OutputFiles, manifest)
	return nil
}

// parseMetafile extracts the island name → output filename manifest from esbuild's metafile.
func (r *renderer) parseMetafile(metafile string, entryBaseToName map[string]string) (map[string]string, error) {
	var meta metafileJSON
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return nil, err
	}

	manifest := make(map[string]string)
	for outPath, output := range meta.Outputs {
		if output.EntryPoint == "" {
			continue // shared chunk, not an island entry
		}
		entryBase := filepath.Base(output.EntryPoint)
		islandName, ok := entryBaseToName[entryBase]
		if !ok {
			continue
		}
		manifest[islandName] = filepath.Base(outPath)
	}
	return manifest, nil
}

// islandCacheDir returns a deterministic cache directory for island assets.
func islandCacheDir(dependencies []string) (string, error) {
	h := sha256.New()
	fmt.Fprintf(h, "dark-islands\n")
	for _, d := range dependencies {
		fmt.Fprintf(h, "%s\n", d)
	}
	hash := hex.EncodeToString(h.Sum(nil))[:12]

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cache", "dark", "islands", hash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (r *renderer) injectIslandsScript(html string) string {
	return insertBeforeTag(html, "</body>", r.getIslandsScriptTag(html))
}

// renderErrorPage renders the configured error component.
func (r *renderer) renderErrorPage(props any) (string, string, error) {
	html, err := r.renderGlobalComponent(globalErrorComponent, props)
	if err != nil {
		return "", "", err
	}
	css := r.collectLayoutCSS(nil) + r.errorCSS
	return html, css, nil
}

// renderNotFoundPage renders the configured not-found component.
func (r *renderer) renderNotFoundPage(props any) (string, string, error) {
	html, err := r.renderGlobalComponent(globalNotFoundComponent, props)
	if err != nil {
		return "", "", err
	}
	css := r.collectLayoutCSS(nil) + r.notFoundCSS
	return html, css, nil
}

// renderGlobalComponent renders a pre-broadcast global component (error/not-found).
func (r *renderer) renderGlobalComponent(globalName string, props any) (string, error) {
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("dark: failed to marshal props: %w", err)
	}

	var code strings.Builder
	fmt.Fprintf(&code, "var __props = %s;\n", propsJSON)
	code.WriteString(r.uikit.rtsResolveJS)
	fmt.Fprintf(&code, "var __Component = globalThis.%s.default || globalThis.%s;\n", globalName, globalName)

	ce := r.uikit.createElement
	if r.hasLayout {
		code.WriteString(layoutResolveJS)
		fmt.Fprintf(&code, "__rts(%s(__Layout, Object.assign({}, __props, { children: %s(__Component, __props) })));\n", ce, ce)
	} else {
		fmt.Fprintf(&code, "__rts(%s(__Component, __props));\n", ce)
	}

	val, err := r.pool.Eval(code.String())
	if err != nil {
		return "", fmt.Errorf("dark: render error page: %w", err)
	}
	defer val.Close()

	html, err := val.GoString()
	if err != nil {
		return "", fmt.Errorf("dark: render error page: failed to get HTML string: %w", err)
	}
	return html, nil
}

// invalidateCache removes a single cache entry by full path and clears the SSR cache.
func (r *renderer) invalidateCache(fullPath string) {
	r.cacheMu.Lock()
	delete(r.cache, fullPath)
	r.cacheMu.Unlock()
	r.clearSSRCache()
}

// invalidateAllCaches clears all cached component bundles.
func (r *renderer) invalidateAllCaches() {
	r.cacheMu.Lock()
	r.cache = make(map[string]*cacheEntry)
	r.cacheMu.Unlock()
}

// invalidateAll clears all component bundle caches and page CSS.
func (r *renderer) invalidateAll() {
	r.invalidateAllCaches()
	r.clearPageCSS()
	r.clearSSRCache()
}

// reloadLayout re-bundles and re-broadcasts the layout component.
func (r *renderer) reloadLayout(cfg *config) error {
	if cfg.layoutFile == "" {
		return nil
	}
	css, err := r.bundleAndBroadcast(cfg.layoutFile, globalLayout)
	if err != nil {
		return err
	}
	r.layoutCSSMu.Lock()
	r.layoutCSS = css
	r.layoutCSSMu.Unlock()
	r.invalidateAll()
	return nil
}

// rebuildClientBundle regenerates the client-side island chunks.
func (r *renderer) rebuildClientBundle(islands []islandEntry, cfg *config) error {
	if !r.hasIslands || r.nodeModulesDir == "" {
		return nil
	}
	if err := r.buildClientBundle(islands, cfg, r.nodeModulesDir); err != nil {
		return fmt.Errorf("dark: failed to rebuild client bundle: %w", err)
	}
	return nil
}

// setClientChunks stores all output files from the code-split build.
func (r *renderer) setClientChunks(outputFiles []api.OutputFile, manifest map[string]string) {
	chunks := make(map[string]string, len(outputFiles))
	for _, f := range outputFiles {
		name := filepath.Base(f.Path)
		chunks[name] = string(f.Contents)
	}
	entryFiles := make(map[string]bool, len(manifest))
	for _, f := range manifest {
		entryFiles[f] = true
	}
	r.clientChunksMu.Lock()
	r.clientChunks = chunks
	r.clientManifest = manifest
	r.clientEntryFiles = entryFiles
	r.clientChunksMu.Unlock()
}

// getClientChunk returns the JS content for a given chunk filename.
func (r *renderer) getClientChunk(filename string) (string, bool) {
	r.clientChunksMu.RLock()
	defer r.clientChunksMu.RUnlock()
	content, ok := r.clientChunks[filename]
	return content, ok
}

// getIslandsScriptTag generates the inline boot script and modulepreload hints
// for islands present in the rendered HTML.
func (r *renderer) getIslandsScriptTag(html string) string {
	r.clientChunksMu.RLock()
	manifest := r.clientManifest
	chunks := r.clientChunks
	entryFiles := r.clientEntryFiles
	r.clientChunksMu.RUnlock()

	if len(manifest) == 0 {
		return ""
	}

	strategies := extractIslandLoadStrategies(html)
	if len(strategies) == 0 {
		return ""
	}

	var sb strings.Builder

	// Modulepreload shared chunks (Preact etc.).
	for filename := range chunks {
		if !entryFiles[filename] {
			fmt.Fprintf(&sb, "<link rel=\"modulepreload\" href=\"/_dark/islands/%s\">\n", filename)
		}
	}

	// Modulepreload eagerly-loaded island chunks.
	for name, strategy := range strategies {
		if strategy == "load" {
			if filename, ok := manifest[name]; ok {
				fmt.Fprintf(&sb, "<link rel=\"modulepreload\" href=\"/_dark/islands/%s\">\n", filename)
			}
		}
	}

	// Build full manifest JSON (all islands, so htmx partials can load any island).
	fullManifest := make(map[string]string, len(manifest))
	for name, filename := range manifest {
		fullManifest[name] = "/_dark/islands/" + filename
	}
	manifestJSON, _ := json.Marshal(fullManifest)

	fmt.Fprintf(&sb, "<script type=\"module\">\n")
	fmt.Fprintf(&sb, "var __dark_manifest = %s;\n", manifestJSON)
	sb.WriteString(bootScriptJS)
	sb.WriteString("</script>")

	return sb.String()
}

// storePageCSS stores CSS content by its content hash and returns the hash.
func (r *renderer) storePageCSS(css string) string {
	h := sha256.Sum256([]byte(css))
	hash := hex.EncodeToString(h[:])[:16]

	r.pageCSSMu.Lock()
	r.pageCSS[hash] = css
	r.pageCSSMu.Unlock()

	return hash
}

// getPageCSS retrieves stored CSS by hash.
func (r *renderer) getPageCSS(hash string) string {
	r.pageCSSMu.RLock()
	defer r.pageCSSMu.RUnlock()
	return r.pageCSS[hash]
}

// clearPageCSS removes all stored page CSS (used in dev mode on changes).
func (r *renderer) clearPageCSS() {
	r.pageCSSMu.Lock()
	r.pageCSS = make(map[string]string)
	r.pageCSSMu.Unlock()
}

// getSourceMap returns the cached source map for a component (dev mode only).
func (r *renderer) getSourceMap(componentPath string) *sourceMap {
	fullPath := filepath.Join(r.templateDir, componentPath)
	r.cacheMu.RLock()
	entry, ok := r.cache[fullPath]
	r.cacheMu.RUnlock()
	if ok && entry.srcMap != nil {
		return entry.srcMap
	}
	return nil
}

// resolveRouteLayout returns the absolute path and entry for a route layout name.
func (r *renderer) resolveRouteLayout(name string) (string, *layoutEntry, bool) {
	absPath, _ := filepath.Abs(filepath.Join(r.templateDir, name))
	r.routeLayoutsMu.RLock()
	entry, ok := r.routeLayouts[absPath]
	r.routeLayoutsMu.RUnlock()
	return absPath, entry, ok
}

// collectLayoutCSS gathers CSS from the global layout and route layouts into a single string.
func (r *renderer) collectLayoutCSS(routeLayouts []string) string {
	var b strings.Builder
	if r.hasLayout {
		r.layoutCSSMu.RLock()
		if r.layoutCSS != "" {
			b.WriteString(r.layoutCSS)
			b.WriteByte('\n')
		}
		r.layoutCSSMu.RUnlock()
	}
	for _, rl := range routeLayouts {
		if _, entry, ok := r.resolveRouteLayout(rl); ok && entry.css != "" {
			b.WriteString(entry.css)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ssrCacheKey builds a cache key for SSR output.
func (r *renderer) ssrCacheKey(componentPath string, layouts []string, skipLayout bool, propsJSON []byte) string {
	h := sha256.New()
	h.Write([]byte(componentPath))
	h.Write([]byte{0})
	for _, l := range layouts {
		h.Write([]byte(l))
		h.Write([]byte{','})
	}
	h.Write([]byte{0})
	if skipLayout {
		h.Write([]byte{'1'})
	} else {
		h.Write([]byte{'0'})
	}
	h.Write([]byte{0})
	h.Write(propsJSON)
	return hex.EncodeToString(h.Sum(nil))
}

func (r *renderer) putSSRCache(key, html, css string) {
	r.ssrCache.put(key, ssrCacheEntry{html: html, css: css})
}

func (r *renderer) clearSSRCache() {
	r.ssrCache.clear()
}

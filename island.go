package dark

import (
	"fmt"
	"regexp"
	"strings"
)

type islandEntry struct {
	name    string
	tsxPath string // relative to templateDir
}

// buildIslandEntryJS generates an ESM entry module for a single island component.
// Each entry exports a default function that hydrates a <dark-island> element.
// absTemplateDir is the absolute path to the template directory, used to construct
// absolute import paths since entry files live in a temp directory.
func buildIslandEntryJS(island islandEntry, absTemplateDir string, kit *uikit, devMode bool) string {
	absIslandPath := absTemplateDir + "/" + island.tsxPath
	var sb strings.Builder
	sb.WriteString(kit.islandImport)
	fmt.Fprintf(&sb, "import __Comp from '%s';\n", absIslandPath)
	sb.WriteString("var C = __Comp.default || __Comp;\n")
	fmt.Fprintf(&sb, `export default function(el) {
  try {
    var props = JSON.parse(el.getAttribute('data-props') || '{}');
    %s
    el.setAttribute('data-hydrated', '');
  } catch (err) {
    console.error('[dark] hydration error for island "' + el.getAttribute('data-name') + '":', err);`, kit.islandHydrate)
	if devMode {
		sb.WriteString(`
    el.innerHTML = '<div style="border:2px solid #e94560;background:#1a1a2e;color:#e94560;padding:12px;font-family:monospace;border-radius:4px;margin:4px 0;">'
      + '<strong>Island Error: ' + el.getAttribute('data-name') + '</strong><br><pre style="margin:8px 0 0;white-space:pre-wrap;">'
      + (err.stack || err.message || String(err)) + '</pre></div>';`)
	}
	sb.WriteString(`
  }
}
`)
	return sb.String()
}

// bootScriptJS is the inline boot script injected into pages that use islands.
// It reads __dark_manifest (island name → chunk URL) and orchestrates lazy loading
// via dynamic import() based on each island's loading strategy.
const bootScriptJS = `
var __loaded = {};

function __dark_doHydrate(el, name, url) {
  if (el.hasAttribute('data-hydrated')) return;
  if (!__loaded[name]) {
    __loaded[name] = import(url);
  }
  __loaded[name].then(function(mod) {
    mod.default(el);
  }).catch(function(err) {
    console.error('[dark] failed to load island "' + name + '":', err);
  });
}

function __dark_hydrateIsland(el) {
  var name = el.getAttribute('data-name');
  var url = __dark_manifest[name];
  if (!url) return;
  var load = el.getAttribute('data-load') || 'load';
  if (load === 'load') {
    __dark_doHydrate(el, name, url);
  } else if (load === 'idle') {
    var cb = function() { __dark_doHydrate(el, name, url); };
    'requestIdleCallback' in window ? requestIdleCallback(cb) : setTimeout(cb, 200);
  } else if (load === 'visible' && 'IntersectionObserver' in window) {
    var obs = new IntersectionObserver(function(entries) {
      entries.forEach(function(e) {
        if (e.isIntersecting) {
          obs.unobserve(e.target);
          __dark_doHydrate(e.target, name, url);
        }
      });
    });
    obs.observe(el);
  } else {
    __dark_doHydrate(el, name, url);
  }
}

function __dark_hydrateAll() {
  document.querySelectorAll('dark-island:not([data-hydrated])').forEach(__dark_hydrateIsland);
}

document.readyState === 'loading'
  ? document.addEventListener('DOMContentLoaded', __dark_hydrateAll)
  : __dark_hydrateAll();

document.addEventListener('htmx:afterSettle', function(evt) {
  var target = evt.detail.elt || document;
  target.querySelectorAll('dark-island:not([data-hydrated])').forEach(__dark_hydrateIsland);
});
`

// islandLoadRe matches data-name and data-load attributes in any order within <dark-island>.
var islandLoadRe = regexp.MustCompile(`<dark-island\b[^>]*\bdata-name="([^"]+)"[^>]*\bdata-load="([^"]+)"`)

// extractIslandLoadStrategies returns a map of island name → loading strategy
// for all islands present in the rendered HTML.
func extractIslandLoadStrategies(html string) map[string]string {
	matches := islandLoadRe.FindAllStringSubmatch(html, -1)
	result := make(map[string]string)
	for _, m := range matches {
		if _, ok := result[m[1]]; !ok {
			result[m[1]] = m[2]
		}
	}
	return result
}

package dark

// reactPolyfillJS provides polyfills for APIs required by React's
// react-dom/server that may not be available in minimal JS runtimes.
const reactPolyfillJS = `
if (typeof MessageChannel === 'undefined') {
  globalThis.MessageChannel = function() {
    var cb = null;
    this.port1 = {};
    this.port2 = {
      postMessage: function() { if (cb) cb({ data: undefined }); }
    };
    Object.defineProperty(this.port1, 'onmessage', {
      set: function(fn) { cb = fn; },
      get: function() { return cb; }
    });
  };
}
if (typeof TextEncoder === 'undefined') {
  globalThis.TextEncoder = function() {};
  globalThis.TextEncoder.prototype.encode = function(str) {
    var arr = [];
    for (var i = 0; i < str.length; i++) {
      var c = str.charCodeAt(i);
      if (c < 0x80) arr.push(c);
      else if (c < 0x800) arr.push(0xC0 | (c >> 6), 0x80 | (c & 0x3F));
      else if (c < 0xD800 || c >= 0xE000) arr.push(0xE0 | (c >> 12), 0x80 | ((c >> 6) & 0x3F), 0x80 | (c & 0x3F));
      else { i++; c = 0x10000 + (((c & 0x3FF) << 10) | (str.charCodeAt(i) & 0x3FF)); arr.push(0xF0 | (c >> 18), 0x80 | ((c >> 12) & 0x3F), 0x80 | ((c >> 6) & 0x3F), 0x80 | (c & 0x3F)); }
    }
    return new Uint8Array(arr);
  };
}
if (typeof TextDecoder === 'undefined') {
  globalThis.TextDecoder = function() {};
  globalThis.TextDecoder.prototype.decode = function(buf) {
    var arr = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
    var s = '', i = 0;
    while (i < arr.length) {
      var b = arr[i++];
      if (b < 0x80) s += String.fromCharCode(b);
      else if (b < 0xE0) s += String.fromCharCode(((b & 0x1F) << 6) | (arr[i++] & 0x3F));
      else if (b < 0xF0) { var c2 = arr[i++]; s += String.fromCharCode(((b & 0x0F) << 12) | ((c2 & 0x3F) << 6) | (arr[i++] & 0x3F)); }
      else { var c2 = arr[i++], c3 = arr[i++]; var cp = ((b & 0x07) << 18) | ((c2 & 0x3F) << 12) | ((c3 & 0x3F) << 6) | (arr[i++] & 0x3F); cp -= 0x10000; s += String.fromCodePoint(cp); }
    }
    return s;
  };
}
if (typeof queueMicrotask === 'undefined') {
  globalThis.queueMicrotask = function(fn) { Promise.resolve().then(fn); };
}
`

// UILibrary selects the JSX library used for SSR and client-side hydration.
type UILibrary int

const (
	// Preact is the default UI library.
	Preact UILibrary = iota
	// React selects React/ReactDOM for SSR and hydration.
	React
)

// uikit holds all library-specific strings used in SSR rendering,
// island hydration, and MCP App bundling.
type uikit struct {
	// preloadJS is JavaScript executed before loading dependency bundles.
	// Used for polyfills required by the UI library (e.g., MessageChannel for React).
	preloadJS string
	// ssrDeps are npm packages passed to ramune.Dependencies for SSR.
	ssrDeps []string
	// ssrExternals are esbuild externals for SSR bundles.
	ssrExternals []string
	// clientPkg are npm packages installed for islands/MCP client bundles.
	clientPkg []string
	// clientPkgCheck is the package directory to stat for existence.
	clientPkgCheck string

	jsxFactory    string // esbuild JSXFactory
	jsxFragment   string // esbuild JSXFragment
	createElement string // SSR codegen: globalThis name (e.g., "preact.h" or "react.createElement")

	rtsResolveJS string // JS snippet to resolve __rts (renderToString) from globals
	darkModuleJS string // SSR-side island wrapper module broadcast to workers

	islandImport  string // client island entry: import statement
	islandHydrate string // client island entry: hydrate call

	mcpImport  string // MCP client entry: import statement
	mcpHydrate string // MCP client entry: hydrate call
	mcpRender  string // MCP client entry: re-render call (updates)
}

func resolveUIKit(lib UILibrary) *uikit {
	switch lib {
	case React:
		return &uikit{
			// React's react-dom/server requires several browser/Node APIs.
			preloadJS:      reactPolyfillJS,
			ssrDeps:        []string{"react", "react-dom", "react-dom/server"},
			ssrExternals:   []string{"react", "react-dom", "react-dom/server"},
			clientPkg:      []string{"react", "react-dom"},
			clientPkgCheck: "react",

			jsxFactory:    "React.createElement",
			jsxFragment:   "React.Fragment",
			createElement: "react.createElement", // lowercase: globalThis.react is set by ramune

			rtsResolveJS: "var __rts = typeof react_dom_server === 'object' ? react_dom_server.renderToString : react_dom_server;\n",

			darkModuleJS: `globalThis.dark = {
  island: function(name, Component, options) {
    var loadStrategy = (options && options.load) || 'load';
    return function(props) {
      return react.createElement('dark-island', {
        'data-name': name,
        'data-props': JSON.stringify(props),
        'data-load': loadStrategy
      }, react.createElement(Component, props));
    };
  }
};`,

			islandImport:  "import React from 'react';\nimport { hydrateRoot } from 'react-dom/client';\n",
			islandHydrate: "hydrateRoot(el, React.createElement(C, props));",

			mcpImport:  "import React from 'react';\nimport { hydrateRoot } from 'react-dom/client';\n",
			mcpHydrate: "var __root = hydrateRoot(app, React.createElement(C, props));",
			mcpRender:  "__root.render(React.createElement(C, props));",
		}
	default: // Preact
		return &uikit{
			ssrDeps:        []string{"preact", "preact-render-to-string"},
			ssrExternals:   []string{"preact", "preact-render-to-string"},
			clientPkg:      []string{"preact"},
			clientPkgCheck: "preact",

			jsxFactory:    "h",
			jsxFragment:   "Fragment",
			createElement: "preact.h",

			rtsResolveJS: "var __rts = typeof preact_render_to_string === 'function' ? preact_render_to_string : preact_render_to_string.renderToString;\n",

			darkModuleJS: `globalThis.dark = {
  island: function(name, Component, options) {
    var loadStrategy = (options && options.load) || 'load';
    return function(props) {
      return preact.h('dark-island', {
        'data-name': name,
        'data-props': JSON.stringify(props),
        'data-load': loadStrategy
      }, preact.h(Component, props));
    };
  }
};`,

			islandImport:  "import { h, hydrate } from 'preact';\n",
			islandHydrate: "hydrate(h(C, props), el);",

			mcpImport:  "import { h, hydrate, render } from 'preact';\n",
			mcpHydrate: "hydrate(h(C, props), app);",
			mcpRender:  "render(h(C, props), app);",
		}
	}
}

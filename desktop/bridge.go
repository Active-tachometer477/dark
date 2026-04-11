package desktop

import (
	"encoding/json"
	"fmt"
)

// bridgeJS is injected into every page via WebView.Init.
// It provides the window.dark API for events and window control.
const bridgeJS = `(function() {
  var listeners = {};
  window.dark = {
    on: function(event, fn) {
      if (!listeners[event]) listeners[event] = [];
      listeners[event].push(fn);
    },
    off: function(event, fn) {
      if (!listeners[event]) return;
      if (!fn) { delete listeners[event]; return; }
      listeners[event] = listeners[event].filter(function(f) { return f !== fn; });
    },
    emit: function(event, data) {
      return __dark_emit(event, JSON.stringify(data !== undefined ? data : null));
    },
    setTitle: function(title) { return __dark_set_title(title); },
    close: function() { return __dark_close(); }
  };
  window.__dark_dispatch = function(event, data) {
    var fns = listeners[event];
    if (!fns) return;
    fns.slice().forEach(function(fn) {
      try { fn(data); } catch(e) { console.error("dark event error:", e); }
    });
  };
})();`

// On registers a handler for events emitted from the frontend via
// window.dark.emit(event, data). Safe to call before or during Run.
func (a *App) On(event string, fn func(data any)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handlers[event] = append(a.handlers[event], fn)
}

// Emit sends a named event with data to the frontend. Listeners
// registered via window.dark.on(event, callback) will be invoked.
// Safe to call from any goroutine.
func (a *App) Emit(event string, data any) {
	wv := a.webview()
	if wv == nil {
		return
	}
	dataBytes, _ := json.Marshal(data)
	js := fmt.Sprintf(`window.__dark_dispatch(%s,%s)`, jsonString(event), string(dataBytes))
	wv.Dispatch(func() { wv.Eval(js) })
}

// setupBridge injects the JS bridge and binds internal functions for
// event dispatch and window control. Called during Run on the UI thread.
func (a *App) setupBridge() {
	a.wv.Init(bridgeJS)

	a.wv.Bind("__dark_emit", func(event, dataJSON string) {
		a.mu.Lock()
		fns := make([]func(data any), len(a.handlers[event]))
		copy(fns, a.handlers[event])
		a.mu.Unlock()

		var data any
		if dataJSON != "" && dataJSON != "null" {
			json.Unmarshal([]byte(dataJSON), &data)
		}
		for _, fn := range fns {
			fn(data)
		}
	})

	a.wv.Bind("__dark_set_title", func(title string) {
		a.wv.Dispatch(func() { a.wv.SetTitle(title) })
	})

	a.wv.Bind("__dark_close", func() {
		a.wv.Terminate()
	})
}

// jsonString returns a JSON-encoded string literal for safe JS embedding.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

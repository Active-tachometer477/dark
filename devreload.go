package dark

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type devReloader struct {
	watcher  *fsnotify.Watcher
	renderer *renderer
	config   *config
	islands  []islandEntry

	subsMu    sync.Mutex
	subs      map[uint64]chan struct{}
	nextSubID uint64

	done chan struct{}
}

func newDevReloader(r *renderer, cfg *config, islands []islandEntry) (*devReloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("dark: failed to create file watcher: %w", err)
	}

	if err := addDirRecursive(watcher, cfg.templateDir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("dark: failed to watch template dir: %w", err)
	}

	d := &devReloader{
		watcher:  watcher,
		renderer: r,
		config:   cfg,
		islands:  islands,
		subs:     make(map[uint64]chan struct{}),
		done:     make(chan struct{}),
	}

	go d.watchLoop()
	cfg.logger.Debug("dev reloader watching", "dir", cfg.templateDir)
	return d, nil
}

func (d *devReloader) watchLoop() {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	pending := make(map[string]struct{})

	for {
		select {
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			// Watch new directories.
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = addDirRecursive(d.watcher, event.Name)
					continue
				}
			}
			if !isWatchedFile(event.Name) {
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				pending[event.Name] = struct{}{}
				timer.Reset(100 * time.Millisecond)
			}

		case <-timer.C:
			if len(pending) == 0 {
				continue
			}
			for path := range pending {
				d.handleChange(path)
			}
			pending = make(map[string]struct{})
			d.notifyBrowsers()

		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			d.config.logger.Warn("watcher error", "error", err)

		case <-d.done:
			return
		}
	}
}

func (d *devReloader) handleChange(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	// Check if it's the layout file.
	if d.config.layoutFile != "" {
		layoutAbs, _ := filepath.Abs(filepath.Join(d.config.templateDir, d.config.layoutFile))
		if absPath == layoutAbs {
			d.config.logger.Debug("layout changed, reloading", "path", path)
			if err := d.renderer.reloadLayout(d.config); err != nil {
				d.config.logger.Error("layout reload error", "error", err)
			}
			return
		}
	}

	// Check if it's a route-specific layout file.
	if d.renderer.isRouteLayout(absPath) {
		d.config.logger.Debug("route layout changed, reloading", "path", path)
		// Find the relative path for the layout.
		relPath, _ := filepath.Rel(d.config.templateDir, absPath)
		if err := d.renderer.reloadRouteLayout(relPath); err != nil {
			d.config.logger.Error("route layout reload error", "error", err)
		}
		return
	}

	// CSS file changed: invalidate everything since we don't track CSS dependencies.
	if filepath.Ext(absPath) == ".css" {
		d.config.logger.Debug("CSS changed, invalidating all caches", "path", path)
		d.renderer.invalidateAll()
		if len(d.islands) > 0 {
			if err := d.renderer.rebuildClientBundle(d.islands, d.config); err != nil {
				d.config.logger.Error("client bundle rebuild error", "error", err)
			}
		}
		return
	}

	// Check if it's an island file.
	for _, isl := range d.islands {
		islAbs, _ := filepath.Abs(filepath.Join(d.config.templateDir, isl.tsxPath))
		if absPath == islAbs {
			d.config.logger.Debug("island changed, rebuilding", "path", path)
			d.renderer.invalidateAllCaches()
			if err := d.renderer.rebuildClientBundle(d.islands, d.config); err != nil {
				d.config.logger.Error("client bundle rebuild error", "error", err)
			}
			return
		}
	}

	// Regular component: invalidate its cache entry.
	d.config.logger.Debug("component changed", "path", path)
	d.renderer.invalidateCache(absPath)
}

func (d *devReloader) subscribe() (uint64, chan struct{}) {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()
	id := d.nextSubID
	d.nextSubID++
	ch := make(chan struct{}, 1)
	d.subs[id] = ch
	return id, ch
}

func (d *devReloader) unsubscribe(id uint64) {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()
	delete(d.subs, id)
}

func (d *devReloader) notifyBrowsers() {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()
	for _, ch := range d.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ServeSSE handles the SSE endpoint for live reload notifications.
func (d *devReloader) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id, ch := d.subscribe()
	defer d.unsubscribe(id)

	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-ch:
			fmt.Fprintf(w, "data: reload\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-d.done:
			return
		}
	}
}

func (d *devReloader) close() {
	select {
	case <-d.done:
		return // already closed
	default:
		close(d.done)
	}
	d.watcher.Close()
}

func addDirRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
}

func isWatchedFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".tsx", ".ts", ".jsx", ".js", ".css":
		return true
	}
	return false
}

func injectDevReloadScript(html string) string {
	return insertBeforeTag(html, "</body>", devReloadScript)
}

const devReloadScript = `<script>(function(){var es=new EventSource('/_dark/reload');var ok=false;es.onmessage=function(e){if(e.data==='connected'){ok=true}else if(e.data==='reload'){location.reload()}};es.onerror=function(){if(ok){es.close();setTimeout(function(){location.reload()},1000)}}})();</script>`

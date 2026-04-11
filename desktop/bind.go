package desktop

import (
	"fmt"

	"github.com/crgimenes/glaze"
)

type pendingBind struct {
	name string
	fn   any
}

type pendingMethods struct {
	prefix string
	obj    any
}

// Bind exposes a Go function as a global JavaScript function.
// The function appears as window.<name>(...) and returns a Promise.
//
// The function may accept JSON-serializable arguments and return:
//   - nothing
//   - a value
//   - an error
//   - (value, error)
//
// Call Bind before Run to register bindings. Calling after Run starts
// applies the binding immediately (thread-safe).
func (a *App) Bind(name string, fn any) error {
	a.mu.Lock()
	wv := a.wv
	if wv == nil {
		a.bindings = append(a.bindings, pendingBind{name, fn})
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()

	return dispatchSync(wv, func() error {
		return wv.Bind(name, fn)
	})
}

// BindMethods exposes all exported methods of obj as global JavaScript
// functions named window.<prefix>_<method_name>().
// Method names are converted to snake_case (e.g., GetUser → get_user).
//
// Methods must follow the same signature rules as Bind.
func (a *App) BindMethods(prefix string, obj any) error {
	a.mu.Lock()
	wv := a.wv
	if wv == nil {
		a.methods = append(a.methods, pendingMethods{prefix, obj})
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()

	return dispatchSync(wv, func() error {
		_, err := glaze.BindMethods(wv, prefix, obj)
		return err
	})
}

// applyBindings drains pending Bind and BindMethods registrations.
// Called during Run on the UI thread, before the event loop starts.
func (a *App) applyBindings() error {
	a.mu.Lock()
	bindings := a.bindings
	methods := a.methods
	a.bindings = nil
	a.methods = nil
	a.mu.Unlock()

	for _, b := range bindings {
		if err := a.wv.Bind(b.name, b.fn); err != nil {
			return fmt.Errorf("desktop: bind %q: %w", b.name, err)
		}
	}
	for _, m := range methods {
		if _, err := glaze.BindMethods(a.wv, m.prefix, m.obj); err != nil {
			return fmt.Errorf("desktop: bind methods %q: %w", m.prefix, err)
		}
	}
	return nil
}

// dispatchSync posts fn to the WebView UI thread and blocks until it completes.
func dispatchSync(wv glaze.WebView, fn func() error) error {
	done := make(chan struct{})
	var err error
	wv.Dispatch(func() {
		err = fn()
		close(done)
	})
	<-done
	return err
}

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
	if a.wv == nil {
		a.bindings = append(a.bindings, pendingBind{name, fn})
		return nil
	}
	var bindErr error
	done := make(chan struct{})
	a.wv.Dispatch(func() {
		bindErr = a.wv.Bind(name, fn)
		close(done)
	})
	<-done
	return bindErr
}

// BindMethods exposes all exported methods of obj as global JavaScript
// functions named window.<prefix>_<method_name>().
// Method names are converted to snake_case (e.g., GetUser → get_user).
//
// Methods must follow the same signature rules as Bind.
func (a *App) BindMethods(prefix string, obj any) error {
	if a.wv == nil {
		a.methods = append(a.methods, pendingMethods{prefix, obj})
		return nil
	}
	var bindErr error
	done := make(chan struct{})
	a.wv.Dispatch(func() {
		_, bindErr = glaze.BindMethods(a.wv, prefix, obj)
		close(done)
	})
	<-done
	return bindErr
}

// applyBindings applies all pending Bind and BindMethods registrations.
// Called during Run on the UI thread, before the event loop starts.
func (a *App) applyBindings() error {
	for _, b := range a.bindings {
		if err := a.wv.Bind(b.name, b.fn); err != nil {
			return fmt.Errorf("desktop: bind %q: %w", b.name, err)
		}
	}
	for _, m := range a.methods {
		if _, err := glaze.BindMethods(a.wv, m.prefix, m.obj); err != nil {
			return fmt.Errorf("desktop: bind methods %q: %w", m.prefix, err)
		}
	}
	return nil
}

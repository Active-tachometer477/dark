package desktop

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"github.com/ncruces/zenity"
)

// dialogOptions is the JSON structure accepted from JavaScript for file dialogs.
type dialogOptions struct {
	Title    string   `json:"title"`
	Filename string   `json:"filename"`
	Filters  []string `json:"filters"` // e.g. ["*.txt", "*.md"]
}

func parseDialogOptions(optsJSON string) dialogOptions {
	var opts dialogOptions
	if optsJSON != "" {
		json.Unmarshal([]byte(optsJSON), &opts)
	}
	return opts
}

func zenityOpts(opts dialogOptions) []zenity.Option {
	var zo []zenity.Option
	if opts.Title != "" {
		zo = append(zo, zenity.Title(opts.Title))
	}
	if opts.Filename != "" {
		zo = append(zo, zenity.Filename(opts.Filename))
	}
	if len(opts.Filters) > 0 {
		zo = append(zo, zenity.FileFilter{Name: "Files", Patterns: opts.Filters})
	}
	return zo
}

// setupFeatures binds built-in desktop features (file dialogs, clipboard,
// notifications) to the WebView. Called during Run on the UI thread.
func (a *App) setupFeatures() {
	// --- File Dialogs ---

	a.wv.Bind("__dark_open_file", func(optsJSON string) (string, error) {
		opts := parseDialogOptions(optsJSON)
		path, err := zenity.SelectFile(zenityOpts(opts)...)
		if err == zenity.ErrCanceled {
			return "", nil
		}
		return path, err
	})

	a.wv.Bind("__dark_open_files", func(optsJSON string) ([]string, error) {
		opts := parseDialogOptions(optsJSON)
		paths, err := zenity.SelectFileMultiple(zenityOpts(opts)...)
		if err == zenity.ErrCanceled {
			return nil, nil
		}
		return paths, err
	})

	a.wv.Bind("__dark_save_file", func(optsJSON string) (string, error) {
		opts := parseDialogOptions(optsJSON)
		zo := zenityOpts(opts)
		zo = append(zo, zenity.ConfirmOverwrite())
		path, err := zenity.SelectFileSave(zo...)
		if err == zenity.ErrCanceled {
			return "", nil
		}
		return path, err
	})

	a.wv.Bind("__dark_pick_folder", func(optsJSON string) (string, error) {
		opts := parseDialogOptions(optsJSON)
		zo := append(zenityOpts(opts), zenity.Directory())
		path, err := zenity.SelectFile(zo...)
		if err == zenity.ErrCanceled {
			return "", nil
		}
		return path, err
	})

	// --- Clipboard ---

	a.wv.Bind("__dark_read_clipboard", func() (string, error) {
		return clipboard.ReadAll()
	})

	a.wv.Bind("__dark_write_clipboard", func(text string) error {
		return clipboard.WriteAll(text)
	})

	// --- Notifications ---

	a.wv.Bind("__dark_notify", func(title, message string) error {
		return beeep.Notify(title, message, "")
	})

	// --- Open External URLs ---

	a.wv.Bind("__dark_open_external", openInBrowser)
}

// openInBrowser opens a URL in the system's default browser.
// Only http/https allowed to prevent file:// or javascript: exploitation.
func openInBrowser(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q: only http and https are allowed", u.Scheme)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // reap child process
	return nil
}

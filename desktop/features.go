package desktop

import (
	"encoding/json"

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
		zo := zenityOpts(opts)
		zo = append(zo, zenity.ShowHidden())
		paths, err := zenity.SelectFileMultiple(zo...)
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
		var zo []zenity.Option
		if opts.Title != "" {
			zo = append(zo, zenity.Title(opts.Title))
		}
		if opts.Filename != "" {
			zo = append(zo, zenity.Filename(opts.Filename))
		}
		path, err := zenity.SelectFile(append(zo, zenity.Directory())...)
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
}

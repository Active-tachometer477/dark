package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type packageConfig struct {
	target string
	name   string
	icon   string
	id     string
	outDir string
	arch   string
}

func cmdPackage(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `Usage: dark package <macos|windows|linux> [flags]

Flags:
  --name string    Application name (default: directory name)
  --icon string    Path to icon file (PNG)
  --id string      Bundle identifier (macOS, default: com.example.<name>)
  --out string     Output directory (default: dist)
  --arch string    Target architecture (default: current)`)
		os.Exit(1)
	}

	target := args[0]
	if target != "macos" && target != "windows" && target != "linux" {
		fatal("unknown target: %s (use macos, windows, or linux)", target)
	}

	fset := flag.NewFlagSet("package", flag.ExitOnError)
	name := fset.String("name", detectProjectName(), "Application name")
	icon := fset.String("icon", "", "Path to icon file (PNG)")
	id := fset.String("id", "", "Bundle identifier (macOS)")
	outDir := fset.String("out", "dist", "Output directory")
	arch := fset.String("arch", runtime.GOARCH, "Target architecture")
	fset.Parse(args[1:])

	cfg := packageConfig{
		target: target,
		name:   *name,
		icon:   *icon,
		id:     *id,
		outDir: *outDir,
		arch:   *arch,
	}
	if cfg.id == "" {
		cfg.id = "com.example." + strings.ToLower(strings.ReplaceAll(cfg.name, " ", "-"))
	}

	switch target {
	case "macos":
		packageMacOS(cfg)
	case "windows":
		packageWindows(cfg)
	case "linux":
		packageLinux(cfg)
	}
}

func detectProjectName() string {
	dir, err := os.Getwd()
	if err != nil {
		return "app"
	}
	return filepath.Base(dir)
}

func packageMacOS(cfg packageConfig) {
	appDir := filepath.Join(cfg.outDir, cfg.name+".app")
	macosDir := filepath.Join(appDir, "Contents", "MacOS")
	resDir := filepath.Join(appDir, "Contents", "Resources")

	safeRemoveAll(appDir)
	for _, d := range []string{macosDir, resDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			fatal("create directory: %v", err)
		}
	}

	// Build binary
	fmt.Printf("Building %s (darwin/%s)...\n", cfg.name, cfg.arch)
	buildBinary("darwin", cfg.arch, "", filepath.Join(macosDir, "app"))

	// Launcher script sets CWD to Resources/ so views/ and public/ are found
	launcher := "#!/bin/bash\nDIR=\"$(dirname \"$0\")\"\ncd \"$DIR/../Resources\"\nexec \"$DIR/app\"\n"
	if err := os.WriteFile(filepath.Join(macosDir, cfg.name), []byte(launcher), 0o755); err != nil {
		fatal("write launcher: %v", err)
	}

	copyAssets(resDir)

	if cfg.icon != "" {
		convertToICNS(cfg.icon, filepath.Join(resDir, "icon.icns"))
	}

	// Info.plist
	writeTemplate("templates/Info.plist.tmpl", filepath.Join(appDir, "Contents", "Info.plist"), struct {
		Name string
		ID   string
	}{cfg.name, cfg.id})

	fmt.Printf("Created %s\n", appDir)
}

func packageWindows(cfg packageConfig) {
	appDir := filepath.Join(cfg.outDir, cfg.name)
	safeRemoveAll(appDir)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		fatal("create directory: %v", err)
	}

	fmt.Printf("Building %s (windows/%s, quickjs)...\n", cfg.name, cfg.arch)
	buildBinary("windows", cfg.arch, "-H windowsgui", filepath.Join(appDir, cfg.name+".exe"), "quickjs")

	copyAssets(appDir)

	if cfg.icon != "" {
		mustCopyFile(cfg.icon, filepath.Join(appDir, "icon.png"))
	}

	fmt.Printf("Created %s\n", appDir)
}

func packageLinux(cfg packageConfig) {
	appDir := filepath.Join(cfg.outDir, cfg.name)
	binName := strings.ToLower(strings.ReplaceAll(cfg.name, " ", "-"))

	safeRemoveAll(appDir)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		fatal("create directory: %v", err)
	}

	fmt.Printf("Building %s (linux/%s)...\n", cfg.name, cfg.arch)
	buildBinary("linux", cfg.arch, "", filepath.Join(appDir, binName))

	copyAssets(appDir)

	if cfg.icon != "" {
		ext := filepath.Ext(cfg.icon)
		mustCopyFile(cfg.icon, filepath.Join(appDir, "icon"+ext))
	}

	writeTemplate("templates/app.desktop.tmpl", filepath.Join(appDir, binName+".desktop"), struct {
		Name string
		Exec string
	}{cfg.name, binName})

	fmt.Printf("Created %s\n", appDir)
}

func safeRemoveAll(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fatal("resolve path: %v", err)
	}
	cwd, _ := os.Getwd()
	rel, err := filepath.Rel(cwd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		fatal("refusing to remove %s: outside working directory", abs)
	}
	os.RemoveAll(abs)
}

func buildBinary(goos, goarch, ldflags, output string, tags ...string) {
	args := []string{"build", "-o", output}
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, ".")

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("go build failed: %v", err)
	}
}

func copyAssets(destDir string) {
	for _, dir := range []string{"views", "public"} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			if err := copyDir(dir, filepath.Join(destDir, dir)); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: copy %s: %v\n", dir, err)
			}
		}
	}
}

func convertToICNS(pngPath, icnsPath string) {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "Warning: .icns conversion requires macOS; skipping icon")
		return
	}

	iconsetDir := icnsPath + ".iconset"
	if err := os.MkdirAll(iconsetDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: icon conversion failed: %v\n", err)
		return
	}
	defer os.RemoveAll(iconsetDir)

	sizes := []struct{ name, size string }{
		{"icon_16x16.png", "16"},
		{"icon_16x16@2x.png", "32"},
		{"icon_32x32.png", "32"},
		{"icon_32x32@2x.png", "64"},
		{"icon_128x128.png", "128"},
		{"icon_128x128@2x.png", "256"},
		{"icon_256x256.png", "256"},
		{"icon_256x256@2x.png", "512"},
		{"icon_512x512.png", "512"},
		{"icon_512x512@2x.png", "1024"},
	}
	for _, s := range sizes {
		cmd := exec.Command("sips", "-z", s.size, s.size, pngPath, "--out", filepath.Join(iconsetDir, s.name))
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: sips resize %s failed: %v\n", s.name, err)
			return
		}
	}

	cmd := exec.Command("iconutil", "-c", "icns", iconsetDir, "-o", icnsPath)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: iconutil failed: %v\n", err)
	}
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

func mustCopyFile(src, dst string) {
	if err := copyFile(src, dst); err != nil {
		fatal("copy %s: %v", src, err)
	}
}

# Deploying Dark Applications

## What gets deployed

A dark application requires:

| Item | Purpose | Required? |
|------|---------|-----------|
| Go binary | Your compiled application | Yes |
| `views/` | TSX templates (compiled at startup by esbuild) | Yes |
| `public/` | Static assets (CSS, images, etc.) | If using `app.Static()` |
| Writable `~/.cache/dark/` | Island npm package cache | If using Islands |
| Network access | npm registry (first startup only, for Preact download) | First run only |

TSX files are **not** embedded in the binary. They are compiled at startup and cached in memory.

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP listen port | `3000` |
| `SESSION_SECRET` | HMAC key for session cookies | Required in production |
| `DARK_DEV` | Set to `1` for dev mode | Off |

## Production checklist

```go
dark.New(
    dark.WithDevMode(false),       // Disable hot reload, minify JS/CSS
    dark.WithSSRCache(1000),       // Cache SSR output
)

dark.Sessions([]byte(secret),
    dark.SessionSecure(true),      // HTTPS-only cookies
)
```

- [ ] `SESSION_SECRET` set to a random 32+ byte hex string
- [ ] `WithDevMode(false)` — disables file watching, enables minification
- [ ] `SessionSecure(true)` — cookies only sent over HTTPS
- [ ] `WithSSRCache(n)` — cache rendered HTML for repeated requests
- [ ] Health check endpoint for load balancers (`/api/health`)

## Docker

```bash
docker build -t myapp .
docker run -p 3000:3000 -e SESSION_SECRET=$(openssl rand -hex 32) myapp
```

See `Dockerfile` in this directory for the multi-stage build.

Key points:
- `CGO_ENABLED=0` disables CGO; ramune's default JSC backend uses purego (no CGO)
- Views and static assets are copied into the image
- `~/.cache/dark/` is writable for island package caching

## Fly.io

```bash
# First deploy
fly launch

# Set secrets
fly secrets set SESSION_SECRET=$(openssl rand -hex 32)

# Deploy
fly deploy
```

See `fly.toml` in this directory.

## Railway / Render / Other

These platforms auto-detect Go apps. Ensure:

1. Build command: `go build -o server ./cmd/server`
2. Start command: `./server`
3. Set `PORT`, `SESSION_SECRET` environment variables
4. Include `views/` and `public/` in the deploy artifact

## Systemd (bare metal / VPS)

```ini
[Unit]
Description=Dark App
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/server
Environment=PORT=3000
Environment=SESSION_SECRET=your-secret-here
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
# Build and deploy
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
scp server views/ public/ yourserver:/opt/myapp/
ssh yourserver 'sudo systemctl restart myapp'
```

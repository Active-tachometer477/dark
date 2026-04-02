import { h } from 'preact';

export default function Layout({ children, _head, user }: any) {
  return (
    <html lang="ja">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{_head?.title || 'Dark App'}</title>
        {_head?.meta?.map((m: any, i: number) =>
          <meta key={i} name={m.name || undefined} property={m.property || undefined} content={m.content} />
        )}
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
      </head>
      <body>
        <header class="site-header">
          <nav>
            <a href="/" class="logo">Dark</a>
            <div class="nav-links">
              <a href="/">Home</a>
              <a href="/blog">Blog</a>
              <a href="/admin/tasks">Tasks</a>
              <a href="/contact">Contact</a>
              <a href="/broken" style="color: #e94560;">Broken</a>
              {user
                ? <form method="POST" action="/logout" style="display:inline;margin:0;">
                    <span style="color:#888;margin-right:8px;">{user}</span>
                    <button type="submit" style="background:none;border:none;color:#e94560;cursor:pointer;font:inherit;">Logout</button>
                  </form>
                : <a href="/login">Login</a>
              }
            </div>
          </nav>
        </header>
        <main>{children}</main>
        <footer class="site-footer">
          <p>Powered by Dark + Preact SSR + htmx</p>
        </footer>
      </body>
    </html>
  );
}

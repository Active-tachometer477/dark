export default function Layout({ children, user }: any) {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>Dark + SQLite Demo</title>
        <style>{`
          body { font-family: system-ui, sans-serif; max-width: 600px; margin: 2rem auto; padding: 0 1rem; color: #333; }
          nav { display: flex; justify-content: space-between; align-items: center; padding: 0.5rem 0; border-bottom: 1px solid #eee; margin-bottom: 1.5rem; }
          nav a { color: #333; text-decoration: none; font-weight: 600; }
          .flash { background: #d4edda; color: #155724; padding: 10px 14px; border-radius: 4px; margin-bottom: 1rem; border: 1px solid #c3e6cb; }
          .btn { padding: 6px 14px; border: 1px solid #ccc; border-radius: 4px; background: #fff; cursor: pointer; font: inherit; }
          .btn:hover { background: #f5f5f5; }
          .btn-primary { background: #333; color: #fff; border-color: #333; }
          .btn-primary:hover { background: #555; }
          .btn-danger { color: #e94560; border-color: #e94560; }
          .btn-danger:hover { background: #fff0f3; }
          input[type="text"] { padding: 8px; border: 1px solid #ccc; border-radius: 4px; font: inherit; width: 100%; }
        `}</style>
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
      </head>
      <body>
        <nav>
          <a href="/">Dark + SQLite</a>
          <div>
            {user
              ? <span>
                  <a href="/todos" style="margin-right: 12px;">{user}'s Todos</a>
                  <form method="POST" action="/logout" style="display:inline;">
                    <button class="btn" type="submit">Logout</button>
                  </form>
                </span>
              : <a href="/login">Login</a>
            }
          </div>
        </nav>
        {children}
      </body>
    </html>
  );
}

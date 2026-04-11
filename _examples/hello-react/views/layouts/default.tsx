export default function Layout({ children, title }: any) {
  return (
    <html lang="en">
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{title || 'Dark + React'}</title>
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
        <link rel="stylesheet" href="/static/style.css" />
      </head>
      <body>
        <header>
          <nav>
            <a href="/" className="logo">Dark + React</a>
            <div>
              <a href="/">Home</a>
              <a href="/about">About</a>
            </div>
          </nav>
        </header>
        <main>{children}</main>
        <footer>
          <p>Powered by Dark + React SSR + htmx</p>
        </footer>
      </body>
    </html>
  );
}

import { h } from 'preact';

export default function Layout({ children, title }) {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{title || 'Dark Showcase'}</title>
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
        <link rel="stylesheet" href="/static/style.css" />
      </head>
      <body>
        <nav>
          <a href="/">Home</a>
          <a href="/dashboard">Dashboard</a>
          <a href="/stats">Stats</a>
          <a href="/contact">Contact</a>
        </nav>
        <main>{children}</main>
        <footer>
          <p>dark framework — Showcase</p>
        </footer>
      </body>
    </html>
  );
}

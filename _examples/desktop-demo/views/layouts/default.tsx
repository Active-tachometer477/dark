export default function Layout({ children, _head }: any) {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{_head?.title || 'Desktop Demo'}</title>
        <style>{`
          * { margin: 0; padding: 0; box-sizing: border-box; }
          body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: #0f0f0f;
            color: #e0e0e0;
            min-height: 100vh;
          }
          main {
            max-width: 640px;
            margin: 0 auto;
            padding: 40px 24px;
          }
          h1 {
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 8px;
            background: linear-gradient(135deg, #7c3aed, #ec4899);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
          }
          .subtitle {
            color: #888;
            margin-bottom: 32px;
            font-size: 14px;
          }
          .section {
            background: #1a1a1a;
            border: 1px solid #2a2a2a;
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 16px;
          }
          .section h2 {
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: #888;
            margin-bottom: 12px;
          }
          button {
            background: #7c3aed;
            color: white;
            border: none;
            border-radius: 8px;
            padding: 8px 16px;
            font-size: 14px;
            cursor: pointer;
            transition: background 0.15s;
            margin-right: 8px;
            margin-bottom: 8px;
          }
          button:hover { background: #6d28d9; }
          button.secondary {
            background: #2a2a2a;
            color: #ccc;
          }
          button.secondary:hover { background: #333; }
          button.danger {
            background: #dc2626;
          }
          button.danger:hover { background: #b91c1c; }
          #log {
            background: #111;
            border: 1px solid #2a2a2a;
            border-radius: 8px;
            padding: 12px;
            font-family: 'SF Mono', 'Fira Code', monospace;
            font-size: 12px;
            line-height: 1.6;
            max-height: 200px;
            overflow-y: auto;
            color: #aaa;
          }
          #log .entry { margin-bottom: 2px; }
          #log .go { color: #7c3aed; }
          #log .js { color: #ec4899; }
          #log .event { color: #22c55e; }
          .result {
            margin-top: 8px;
            padding: 8px 12px;
            background: #111;
            border-radius: 6px;
            font-family: monospace;
            font-size: 13px;
            color: #7c3aed;
          }
        `}</style>
      </head>
      <body>
        <main>{children}</main>
      </body>
    </html>
  );
}

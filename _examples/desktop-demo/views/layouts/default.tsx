export default function Layout({ children, _head }: any) {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{_head?.title || 'Dark Notes'}</title>
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
        <style>{`
          * { margin: 0; padding: 0; box-sizing: border-box; }
          body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: #0a0a0a;
            color: #e0e0e0;
            min-height: 100vh;
          }
          main {
            max-width: 720px;
            margin: 0 auto;
            padding: 32px 24px;
          }
          h1 {
            font-size: 24px;
            font-weight: 700;
            background: linear-gradient(135deg, #7c3aed, #ec4899);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
          }
          .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid #1a1a1a;
          }
          .header-right {
            display: flex;
            align-items: center;
            gap: 12px;
          }
          .flash {
            background: #065f46;
            color: #a7f3d0;
            padding: 10px 16px;
            border-radius: 8px;
            margin-bottom: 16px;
            font-size: 13px;
            border: 1px solid #047857;
          }
          .card {
            background: #141414;
            border: 1px solid #1e1e1e;
            border-radius: 10px;
            padding: 16px;
            margin-bottom: 12px;
          }
          .card h3 {
            font-size: 13px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: #666;
            margin-bottom: 10px;
          }
          .note {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            padding: 10px 12px;
            background: #0f0f0f;
            border: 1px solid #1e1e1e;
            border-radius: 6px;
            margin-bottom: 6px;
          }
          .note-title { font-weight: 600; font-size: 14px; }
          .note-body { color: #888; font-size: 12px; margin-top: 2px; }
          .note-time { color: #555; font-size: 11px; margin-top: 4px; }
          .note-delete {
            background: none;
            border: none;
            color: #555;
            cursor: pointer;
            font-size: 16px;
            padding: 0 4px;
            line-height: 1;
          }
          .note-delete:hover { color: #ef4444; }
          .empty { color: #555; font-size: 13px; text-align: center; padding: 20px; }
          input, textarea {
            width: 100%;
            background: #0f0f0f;
            border: 1px solid #2a2a2a;
            border-radius: 6px;
            padding: 8px 10px;
            color: #e0e0e0;
            font-family: inherit;
            font-size: 13px;
            outline: none;
          }
          input:focus, textarea:focus { border-color: #7c3aed; }
          textarea { resize: vertical; min-height: 60px; }
          .form-row { margin-bottom: 8px; }
          .field-error { color: #ef4444; font-size: 12px; margin-top: 4px; }
          .has-error input, .has-error textarea { border-color: #ef4444; }
          label { font-size: 12px; color: #888; margin-bottom: 4px; display: block; }
          button, .btn {
            background: #7c3aed;
            color: white;
            border: none;
            border-radius: 6px;
            padding: 7px 14px;
            font-size: 13px;
            cursor: pointer;
            transition: background 0.15s;
            text-decoration: none;
            display: inline-block;
          }
          button:hover, .btn:hover { background: #6d28d9; }
          .btn-sm { padding: 5px 10px; font-size: 12px; }
          .btn-ghost {
            background: #1a1a1a;
            color: #aaa;
          }
          .btn-ghost:hover { background: #222; color: #fff; }
          .btn-danger { background: #dc2626; }
          .btn-danger:hover { background: #b91c1c; }
          .actions { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 10px; }
          .info-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 6px;
          }
          .info-item {
            padding: 6px 10px;
            background: #0f0f0f;
            border-radius: 4px;
            font-size: 12px;
          }
          .info-label { color: #666; }
          .info-value { color: #7c3aed; font-family: 'SF Mono', monospace; }
          .username-form {
            display: flex;
            gap: 6px;
            align-items: center;
          }
          .username-form input { width: 120px; padding: 5px 8px; font-size: 12px; }
          .user-badge {
            background: #1a1a1a;
            color: #7c3aed;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
          }
          .toast {
            position: fixed;
            top: 16px;
            right: 16px;
            background: #065f46;
            color: #a7f3d0;
            padding: 10px 16px;
            border-radius: 8px;
            font-size: 13px;
            border: 1px solid #047857;
            z-index: 1000;
            animation: fadeIn 0.3s ease;
          }
          @keyframes fadeIn { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }
        `}</style>
      </head>
      <body>
        <main>{children}</main>
      </body>
    </html>
  );
}

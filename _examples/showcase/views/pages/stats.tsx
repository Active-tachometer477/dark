import { h } from 'preact';

// Demonstrates: SSR Cache + ETag / 304 Not Modified
//
// This page is served from the LRU SSR cache (WithSSRCache(100)).
// On subsequent requests with the same props, the server returns a
// cached response with an ETag header. If the browser sends
// If-None-Match, the server responds with 304 Not Modified.

export default function StatsPage({ visitors, pageViews, signups }) {
  return (
    <div>
      <dark-head>
        <title>Stats — SSR Cache + ETag</title>
      </dark-head>

      <h1>Stats</h1>
      <p class="hint">
        This page uses <code>WithSSRCache(100)</code> for LRU caching and
        returns an <code>ETag</code> header. Refresh to see 304 Not Modified
        in DevTools Network tab.
      </p>

      <div class="grid">
        <div class="stat-card">
          <span class="stat-value">{visitors?.toLocaleString()}</span>
          <span class="stat-label">Visitors</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{pageViews?.toLocaleString()}</span>
          <span class="stat-label">Page Views</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{signups?.toLocaleString()}</span>
          <span class="stat-label">Signups</span>
        </div>
      </div>

      <p>
        <button hx-get="/stats" hx-target="main" hx-swap="innerHTML">
          Refresh (htmx partial — no layout, no ETag)
        </button>
      </p>
    </div>
  );
}

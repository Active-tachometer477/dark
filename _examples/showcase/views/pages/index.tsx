// Demonstrates:
//   - Context.Set / Context.Get (requestID from middleware)
//   - CSRF meta tag auto-injection (visible in page source)
//   - _csrfToken in props (available for forms)

export default function IndexPage({ title, requestID, _csrfToken }) {
  return (
    <div>
      <dark-head>
        <title>{title}</title>
      </dark-head>

      <h1>{title}</h1>

      <section class="feature-card">
        <h2>Context.Set / Context.Get</h2>
        <p>Request ID from middleware: <code>{requestID}</code></p>
        <p>
          The <code>requestIDMiddleware</code> calls{' '}
          <code>dark.SetValue(r, "requestID", id)</code>, and the Loader
          retrieves it via <code>ctx.Get("requestID")</code>.
        </p>
      </section>

      <section class="feature-card">
        <h2>CSRF Protection</h2>
        <p>CSRF token in props: <code>{_csrfToken ? _csrfToken.substring(0, 16) + '...' : 'N/A'}</code></p>
        <p>
          View page source to see the auto-injected{' '}
          <code>&lt;meta name="csrf-token"&gt;</code> and htmx config script.
        </p>
      </section>

      <section class="feature-card">
        <h2>New Features</h2>
        <ul>
          <li><a href="/dashboard">Concurrent Loaders</a> — 3 data sources in ~50ms</li>
          <li><a href="/stats">SSR Cache + ETag</a> — 304 Not Modified support</li>
          <li><a href="/contact">CSRF Form</a> — htmx auto-injection</li>
        </ul>
      </section>
    </div>
  );
}

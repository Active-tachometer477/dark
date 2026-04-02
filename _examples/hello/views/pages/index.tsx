import { h } from 'preact';
import { island } from 'dark';
import Counter from '../islands/counter.tsx';
import './style.css';

const InteractiveCounter = island('counter', Counter);

export default function IndexPage({ message, count, user, flashes }: any) {
  return (
    <div>
      {flashes?.notice && (
        <div style="background: #d4edda; color: #155724; padding: 12px 16px; border-radius: 4px; margin-bottom: 1rem; border: 1px solid #c3e6cb;">
          {flashes.notice}
        </div>
      )}
      <div class="hero">
        <h1>{message}</h1>
        <p>A Go SSR framework with Preact, htmx, and Islands architecture</p>
      </div>

      <div class="features">
        <div class="feature-card">
          <h3>Nested Layouts</h3>
          <p>Route-specific and group layouts that nest inside the global layout. Check out the <a href="/admin/tasks">admin panel</a>.</p>
        </div>
        <div class="feature-card">
          <h3>Form Validation</h3>
          <p>Server-side validation with automatic error feedback. Try the <a href="/contact">contact form</a>.</p>
        </div>
        <div class="feature-card">
          <h3>Head Management</h3>
          <p>Per-page title and meta tags via context or {'<dark-head>'}. See it on <a href="/blog">blog posts</a>.</p>
        </div>
        <div class="feature-card">
          <h3>Streaming SSR</h3>
          <p>Shell-first rendering sends {'<head>'} immediately for faster TTFB. Enabled on the <a href="/blog">blog</a>.</p>
        </div>
      </div>

      <div class="demo-section">
        <h2>Interactive Island</h2>
        <p style="margin-bottom: 1rem; color: #666;">This counter is server-rendered, then hydrated on the client.</p>
        <InteractiveCounter initialCount={count} />
      </div>
    </div>
  );
}

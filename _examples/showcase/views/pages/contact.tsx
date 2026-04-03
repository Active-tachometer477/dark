import { h } from 'preact';

// Demonstrates: CSRF protection with htmx
//
// The CSRF middleware auto-injects:
//   1. <meta name="csrf-token"> in <head>
//   2. htmx:configRequest script that attaches X-CSRF-Token header
//   3. _csrfToken in Loader props for hidden form fields
//
// Both htmx requests (via auto-header) and regular form submissions
// (via hidden field) are protected.

export default function ContactPage({ success, _csrfToken, _errors, _formData }) {
  if (success) {
    return (
      <div>
        <dark-head><title>Contact — Sent!</title></dark-head>
        <h1>Contact</h1>
        <div class="success">Message sent successfully!</div>
        <a href="/contact">Send another</a>
      </div>
    );
  }

  const errors = _errors || [];
  const formData = _formData || {};
  const errorFor = (field) => {
    const e = errors.find(e => e.field === field);
    return e ? <span class="field-error">{e.message}</span> : null;
  };

  return (
    <div>
      <dark-head><title>Contact — CSRF Demo</title></dark-head>

      <h1>Contact</h1>
      <p class="hint">
        This form is CSRF-protected. The <code>X-CSRF-Token</code> header is
        auto-attached to htmx requests. A hidden <code>_csrf</code> field
        protects regular form submissions.
      </p>

      {/* Regular form with hidden CSRF field */}
      <form method="POST" action="/contact">
        <input type="hidden" name="_csrf" value={_csrfToken} />

        <label>
          Name
          <input type="text" name="name" value={formData.name || ''} />
          {errorFor('name')}
        </label>

        <label>
          Message
          <textarea name="message" rows="4">{formData.message || ''}</textarea>
          {errorFor('message')}
        </label>

        <button type="submit">Send (regular form)</button>
      </form>

      <hr />

      {/* htmx form — CSRF header auto-injected by the middleware script */}
      <form hx-post="/contact" hx-target="main" hx-swap="innerHTML">
        <label>
          Name
          <input type="text" name="name" value={formData.name || ''} />
          {errorFor('name')}
        </label>

        <label>
          Message
          <textarea name="message" rows="4">{formData.message || ''}</textarea>
          {errorFor('message')}
        </label>

        <button type="submit">Send (htmx — auto CSRF header)</button>
      </form>
    </div>
  );
}

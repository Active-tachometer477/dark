import './style.css';

export default function ContactPage({ _errors, _formData, success }: any) {
  const getError = (field: string) => _errors?.find((e: any) => e.field === field);
  const nameErr = getError('name');
  const emailErr = getError('email');
  const messageErr = getError('message');

  return (
    <div style="max-width: 600px;">
      <dark-head>
        <title>Contact | Dark App</title>
        <meta name="description" content="Get in touch with us" />
      </dark-head>

      <h1 style="margin-bottom: 1.5rem;">Contact Us</h1>

      {success && (
        <div class="form-success">
          Thank you, {success}! Your message has been sent.
        </div>
      )}

      <div id="contact-form">
        <form hx-post="/contact" hx-target="#contact-form" hx-swap="innerHTML">
          <div class={`form-group ${nameErr ? 'has-error' : ''}`}>
            <label for="name">Name</label>
            <input type="text" id="name" name="name" value={_formData?.name || ''} placeholder="Your name" />
            {nameErr && <div class="field-error">{nameErr.message}</div>}
          </div>

          <div class={`form-group ${emailErr ? 'has-error' : ''}`}>
            <label for="email">Email</label>
            <input type="email" id="email" name="email" value={_formData?.email || ''} placeholder="you@example.com" />
            {emailErr && <div class="field-error">{emailErr.message}</div>}
          </div>

          <div class={`form-group ${messageErr ? 'has-error' : ''}`}>
            <label for="message">Message</label>
            <textarea id="message" name="message" placeholder="What would you like to say?">{_formData?.message || ''}</textarea>
            {messageErr && <div class="field-error">{messageErr.message}</div>}
          </div>

          <button type="submit" class="btn-submit">Send Message</button>
        </form>
      </div>
    </div>
  );
}

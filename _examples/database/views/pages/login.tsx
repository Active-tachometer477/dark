import { h } from 'preact';

export default function Login({ _errors, _formData }: any) {
  const errors = _errors || [];
  const fieldError = (name: string) => errors.find((e: any) => e.field === name);

  return (
    <div>
      <h1>Login</h1>
      <p style="color: #666;">Try <strong>alice</strong> (has seeded todos) or any name.</p>
      <form method="POST" action="/login" style="margin-top: 1rem;">
        <div style="margin-bottom: 1rem;">
          <input type="text" name="username" placeholder="Username" value={_formData?.username || ''}
            style={fieldError('username') ? 'border-color: #e94560;' : ''} />
          {fieldError('username') && <p style="color: #e94560; margin: 4px 0 0; font-size: 0.875rem;">{fieldError('username').message}</p>}
        </div>
        <button class="btn btn-primary" type="submit">Login</button>
      </form>
    </div>
  );
}

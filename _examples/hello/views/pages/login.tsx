export default function LoginPage({ _errors, _formData }: any) {
  const errors = _errors || [];
  const fieldError = (name: string) => errors.find((e: any) => e.field === name);

  return (
    <div style="max-width: 400px; margin: 2rem auto;">
      <h1>Login</h1>
      <p style="color: #666; margin-bottom: 1.5rem;">
        Enter any username to try the session demo.
      </p>

      <form method="POST" action="/login">
        <div style="margin-bottom: 1rem;">
          <label for="username" style="display: block; margin-bottom: 4px; font-weight: 600;">Username</label>
          <input
            type="text"
            id="username"
            name="username"
            value={_formData?.username || ''}
            placeholder="e.g. alice"
            style={`width: 100%; padding: 8px; border: 1px solid ${fieldError('username') ? '#e94560' : '#ccc'}; border-radius: 4px; font-size: 1rem;`}
          />
          {fieldError('username') && (
            <p style="color: #e94560; margin: 4px 0 0; font-size: 0.875rem;">
              {fieldError('username').message}
            </p>
          )}
        </div>

        <button
          type="submit"
          style="width: 100%; padding: 10px; background: #333; color: #fff; border: none; border-radius: 4px; font-size: 1rem; cursor: pointer;"
        >
          Login
        </button>
      </form>
    </div>
  );
}

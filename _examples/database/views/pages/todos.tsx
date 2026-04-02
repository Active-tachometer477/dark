import { h } from 'preact';

export default function Todos({ todos, user, flashes, _errors, _formData }: any) {
  const errors = _errors || [];
  const fieldError = (name: string) => errors.find((e: any) => e.field === name);
  const items = todos || [];

  return (
    <div>
      {flashes?.notice && <div class="flash">{flashes.notice}</div>}

      <h1>{user}'s Todos</h1>

      {/* Add form */}
      <form method="POST" action="/todos" style="display: flex; gap: 8px; margin-bottom: 1.5rem;">
        <div style="flex: 1;">
          <input type="text" name="title" placeholder="What needs to be done?" value={_formData?.title || ''}
            style={fieldError('title') ? 'border-color: #e94560;' : ''} />
          {fieldError('title') && <p style="color: #e94560; margin: 4px 0 0; font-size: 0.875rem;">{fieldError('title').message}</p>}
        </div>
        <button class="btn btn-primary" type="submit">Add</button>
      </form>

      {/* Todo list */}
      {items.length === 0
        ? <p style="color: #999;">No todos yet. Add one above!</p>
        : <ul style="list-style: none; padding: 0;">
            {items.map((t: any) => (
              <li key={t.id} style="display: flex; align-items: center; gap: 8px; padding: 8px 0; border-bottom: 1px solid #eee;">
                <form method="POST" action={`/todos/${t.id}/toggle`} style="display: contents;">
                  <button type="submit" class="btn" style="padding: 2px 8px;">
                    {t.done ? '✓' : '○'}
                  </button>
                </form>
                <span style={t.done ? 'flex:1; text-decoration: line-through; color: #999;' : 'flex:1;'}>
                  {t.title}
                </span>
                <form method="POST" action={`/todos/${t.id}`}
                  hx-delete={`/todos/${t.id}`} hx-target="body" style="display: contents;">
                  <input type="hidden" name="_method" value="DELETE" />
                  <button type="submit" class="btn btn-danger" style="padding: 2px 8px;">×</button>
                </form>
              </li>
            ))}
          </ul>
      }

      <p style="margin-top: 2rem; color: #999; font-size: 0.875rem;">
        {items.length} todo{items.length !== 1 ? 's' : ''} · {items.filter((t: any) => t.done).length} done
      </p>
    </div>
  );
}

import { h } from 'preact';
import './style.css';

export default function AdminTasks({ tasks, _errors, _formData }: any) {
  const titleErr = _errors?.find((e: any) => e.field === 'title');

  return (
    <div id="tasks-page">
      <h1 style="margin-bottom: 1.5rem;">Task Manager</h1>

      <div class="demo-section" style="margin-bottom: 1.5rem;">
        <h2>Add Task</h2>
        <form hx-post="/admin/tasks" hx-target="#tasks-page" hx-swap="outerHTML">
          <div style="display: flex; gap: 0.75rem; align-items: flex-start;">
            <div class={`form-group ${titleErr ? 'has-error' : ''}`} style="flex: 1; margin-bottom: 0;">
              <input type="text" name="title" value={_formData?.title || ''} placeholder="New task..." />
              {titleErr && <div class="field-error">{titleErr.message}</div>}
            </div>
            <select name="priority" style="padding: 0.5rem; border: 1px solid #ddd; border-radius: 6px;">
              <option value="low">Low</option>
              <option value="medium" selected>Medium</option>
              <option value="high">High</option>
            </select>
            <button type="submit" class="btn-submit">Add</button>
          </div>
        </form>
      </div>

      <ul class="task-list">
        {tasks?.map((task: any) => (
          <li key={task.id} class={`task-item ${task.done ? 'done' : ''}`}>
            <span class="task-check">{task.done ? '\u2611' : '\u2610'}</span>
            <span class="task-title">{task.title}</span>
            <span class={`task-priority ${task.priority}`}>{task.priority}</span>
            <div class="task-actions">
              <button
                class="btn-toggle"
                hx-post={`/admin/tasks/${task.id}/toggle`}
                hx-target="#tasks-page"
                hx-swap="outerHTML"
              >
                {task.done ? 'Undo' : 'Done'}
              </button>
              <button
                class="btn-delete"
                hx-delete={`/admin/tasks/${task.id}`}
                hx-target="#tasks-page"
                hx-swap="outerHTML"
              >
                Del
              </button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

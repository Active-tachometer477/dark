import { h } from 'preact';

export default function AdminLayout({ children }: any) {
  return (
    <div class="admin-shell">
      <aside class="admin-sidebar">
        <h3>Admin</h3>
        <ul>
          <li><a href="/admin/tasks">Tasks</a></li>
          <li><a href="/admin/settings">Settings</a></li>
        </ul>
      </aside>
      <div class="admin-main">{children}</div>
    </div>
  );
}

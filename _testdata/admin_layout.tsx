import { h } from 'preact';

export default function AdminLayout({ children, title }: any) {
  return (
    <div class="admin-layout">
      <nav class="admin-nav">Admin Panel</nav>
      <div class="admin-content">{children}</div>
    </div>
  );
}

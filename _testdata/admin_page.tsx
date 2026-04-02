import { h } from 'preact';

export default function AdminPage({ section }: any) {
  return <div class="admin-page">Admin: {section || 'home'}</div>;
}

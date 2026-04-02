import { h } from 'preact';
import './style.css';

export default function AdminSettings({}: any) {
  return (
    <div class="settings-panel">
      <dark-head>
        <title>Settings | Admin | Dark App</title>
      </dark-head>

      <h1>Settings</h1>
      <p style="color: #666;">This page demonstrates a deeply nested layout:</p>
      <ul style="margin: 1rem 0; padding-left: 1.5rem; color: #555;">
        <li>Global Layout (header, footer)</li>
        <li>Admin Layout (sidebar + content area)</li>
        <li>This page content</li>
      </ul>
      <p style="color: #666;">
        Check the page source to see how all three layers compose together.
      </p>
    </div>
  );
}

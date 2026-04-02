import { h } from 'preact';

export default function SettingsLayout({ children }: any) {
  return (
    <div class="settings-layout">
      <aside class="settings-sidebar">Settings Menu</aside>
      <div class="settings-body">{children}</div>
    </div>
  );
}

import { h } from 'preact';

// Demonstrates: Route.Loaders — concurrent data loading
//
// Three independent data sources (user, activity, notifications) are fetched
// in parallel via Route.Loaders. Each simulates 50ms latency, but total
// page load is ~50ms instead of ~150ms sequential.

export default function DashboardPage({ user, activity, notifications }) {
  const unreadCount = (notifications || []).filter(n => n.unread).length;

  return (
    <div>
      <dark-head>
        <title>Dashboard — {user?.name}</title>
      </dark-head>

      <h1>Dashboard</h1>
      <p class="hint">
        This page uses <code>Route.Loaders</code> to fetch 3 data sources
        concurrently (~50ms total instead of ~150ms sequential).
      </p>

      <div class="grid">
        <section class="feature-card">
          <h2>Profile</h2>
          <div class="profile">
            <img src={user?.avatar} alt={user?.name} width="60" height="60" />
            <div>
              <strong>{user?.name}</strong>
              <p>{user?.email}</p>
            </div>
          </div>
        </section>

        <section class="feature-card">
          <h2>Notifications ({unreadCount} unread)</h2>
          <ul>
            {(notifications || []).map((n, i) => (
              <li key={i} class={n.unread ? 'unread' : ''}>
                {n.unread ? '🔴 ' : '⚪ '}{n.message}
              </li>
            ))}
          </ul>
        </section>

        <section class="feature-card">
          <h2>Recent Activity</h2>
          <ul>
            {(activity || []).map((a, i) => (
              <li key={i}>
                <strong>{a.action}</strong> — {a.time}
              </li>
            ))}
          </ul>
        </section>
      </div>
    </div>
  );
}

import { h } from 'preact';

export default function Home({ user, flashes }: any) {
  return (
    <div>
      {flashes?.notice && <div class="flash">{flashes.notice}</div>}
      <h1>Dark + SQLite Demo</h1>
      <p>A minimal todo app showing how to use dark with a relational database.</p>
      <h2>Patterns demonstrated</h2>
      <ul>
        <li>Pass <code>*sql.DB</code> to handlers via closures</li>
        <li><strong>Loader</strong> → <code>SELECT</code> (read)</li>
        <li><strong>Action</strong> → <code>INSERT / UPDATE / DELETE</code> (write)</li>
        <li>Session auth guards DB routes via <code>Group.Use(RequireAuth())</code></li>
        <li>Per-user data isolation (<code>WHERE owner = ?</code>)</li>
      </ul>
      {user
        ? <p><a href="/todos">Go to your todos →</a></p>
        : <p><a href="/login">Login to get started →</a></p>
      }
    </div>
  );
}

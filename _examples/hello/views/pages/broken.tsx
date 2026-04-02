import { h } from 'preact';

export default function BrokenPage({ items }: any) {
  // This will throw a TypeError because items is null
  const count = items.length;
  const names = items.map((i: any) => i.name);

  return (
    <div>
      <h1>This page is intentionally broken</h1>
      <p>Count: {count}</p>
      <ul>
        {names.map((n: string) => <li>{n}</li>)}
      </ul>
    </div>
  );
}

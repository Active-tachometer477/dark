import { h } from 'preact';
import { useState } from 'preact/hooks';

export default function Counter({ initialCount }: any) {
  const [count, setCount] = useState(initialCount || 0);
  return (
    <div style="display: inline-flex; align-items: center; gap: 8px; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
      <button onClick={() => setCount((c: number) => c - 1)} style="padding: 4px 12px; cursor: pointer;">-</button>
      <span style="min-width: 40px; text-align: center; font-size: 1.2em; font-weight: bold;">{count}</span>
      <button onClick={() => setCount((c: number) => c + 1)} style="padding: 4px 12px; cursor: pointer;">+</button>
    </div>
  );
}

import { h } from 'preact';
import { useState } from 'preact/hooks';

export default function Counter({ initial }: any) {
  const [count, setCount] = useState(initial || 0);
  return (
    <div class="counter">
      <span>{count}</span>
      <button onClick={() => setCount((c: number) => c + 1)}>+</button>
    </div>
  );
}

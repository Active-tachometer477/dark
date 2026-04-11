import { useState } from 'react';

export default function Counter({ initialCount }: any) {
  const [count, setCount] = useState(initialCount || 0);
  return (
    <div className="counter">
      <button onClick={() => setCount((c: number) => c - 1)}>-</button>
      <span className="count">{count}</span>
      <button onClick={() => setCount((c: number) => c + 1)}>+</button>
    </div>
  );
}

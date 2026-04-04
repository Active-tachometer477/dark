import React, { useState } from 'react';

export default function Counter({ initial }: any) {
  const [count, setCount] = useState(initial || 0);
  return (
    <div className="counter">
      <span>{count}</span>
      <button onClick={() => setCount((c: number) => c + 1)}>+</button>
    </div>
  );
}

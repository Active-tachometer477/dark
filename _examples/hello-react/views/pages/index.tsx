import { island } from 'dark';
import Counter from '../islands/counter.tsx';

const InteractiveCounter = island('counter', Counter);

export default function IndexPage({ title, message, count }: any) {
  return (
    <div>
      <div className="hero">
        <h1>{title}</h1>
        <p>{message}</p>
      </div>

      <div className="demo">
        <h2>Interactive Island</h2>
        <p>This counter is server-rendered with React, then hydrated on the client via hydrateRoot.</p>
        <InteractiveCounter initialCount={count} />
      </div>

      <div className="demo">
        <h2>htmx Partial Update</h2>
        <p>The counter above is an Island — click +/- to verify client-side hydration works.</p>
      </div>
    </div>
  );
}

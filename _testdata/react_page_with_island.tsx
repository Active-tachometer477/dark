import { island } from 'dark';
import Counter from './react_island_counter.tsx';

const InteractiveCounter = island('counter', Counter);

export default function Page({ title, count }: any) {
  return (
    <div>
      <h1>{title}</h1>
      <InteractiveCounter initial={count} />
    </div>
  );
}

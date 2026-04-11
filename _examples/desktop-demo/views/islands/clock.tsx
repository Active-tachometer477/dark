import { useState, useEffect } from 'preact/hooks';

export default function Clock({ label }: any) {
  const [time, setTime] = useState(new Date().toLocaleTimeString());
  const [running, setRunning] = useState(true);

  useEffect(() => {
    if (!running) return;
    const id = setInterval(() => setTime(new Date().toLocaleTimeString()), 1000);
    return () => clearInterval(id);
  }, [running]);

  return (
    <div style="display: flex; align-items: center; gap: 12px;">
      <div style="font-family: 'SF Mono', 'Fira Code', monospace; font-size: 20px; color: #7c3aed; font-weight: 700; letter-spacing: 0.05em;">
        {time}
      </div>
      <div style="font-size: 12px; color: #666;">{label || 'Local Time'}</div>
      <button
        onClick={() => setRunning(r => !r)}
        style={`background: ${running ? '#1a1a1a' : '#7c3aed'}; border: 1px solid ${running ? '#333' : '#7c3aed'}; color: ${running ? '#888' : '#fff'}; border-radius: 4px; padding: 3px 8px; font-size: 11px; cursor: pointer;`}
      >
        {running ? 'Pause' : 'Resume'}
      </button>
    </div>
  );
}

import { h } from 'preact';
import { useState } from 'preact/hooks';

interface Props {
  period: string;
  visitors: number;
  pageViews: number;
  topPages: Array<{ path: string; views: number }>;
}

export default function Dashboard({ period, visitors, pageViews, topPages }: Props) {
  const [tab, setTab] = useState<'overview' | 'pages'>('overview');

  const btnStyle = (active: boolean) => ({
    padding: '8px 16px',
    border: 'none',
    borderBottom: active ? '2px solid var(--mcp-ui-text-primary, #333)' : '2px solid transparent',
    background: 'none',
    cursor: 'pointer',
    fontWeight: active ? 'bold' : 'normal',
    color: 'inherit',
  });

  return (
    <div style={{ padding: '16px', maxWidth: '600px' }}>
      <h1 style={{ margin: '0 0 4px' }}>Analytics Dashboard</h1>
      <p style={{ color: '#666', margin: '0 0 16px' }}>Period: {period}</p>

      <nav style={{ borderBottom: '1px solid #eee', marginBottom: '16px' }}>
        <button style={btnStyle(tab === 'overview')} onClick={() => setTab('overview')}>
          Overview
        </button>
        <button style={btnStyle(tab === 'pages')} onClick={() => setTab('pages')}>
          Top Pages
        </button>
      </nav>

      {tab === 'overview' && (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
          <div style={{ padding: '16px', background: '#f8f9fa', borderRadius: '8px' }}>
            <div style={{ fontSize: '24px', fontWeight: 'bold' }}>
              {(visitors || 0).toLocaleString()}
            </div>
            <div style={{ color: '#666', fontSize: '14px' }}>Visitors</div>
          </div>
          <div style={{ padding: '16px', background: '#f8f9fa', borderRadius: '8px' }}>
            <div style={{ fontSize: '24px', fontWeight: 'bold' }}>
              {(pageViews || 0).toLocaleString()}
            </div>
            <div style={{ color: '#666', fontSize: '14px' }}>Page Views</div>
          </div>
        </div>
      )}

      {tab === 'pages' && (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '1px solid #eee' }}>
                Page
              </th>
              <th style={{ textAlign: 'right', padding: '8px', borderBottom: '1px solid #eee' }}>
                Views
              </th>
            </tr>
          </thead>
          <tbody>
            {(topPages || []).map((page, i) => (
              <tr key={i}>
                <td style={{ padding: '8px', borderBottom: '1px solid #f0f0f0' }}>
                  {page.path}
                </td>
                <td style={{ textAlign: 'right', padding: '8px', borderBottom: '1px solid #f0f0f0' }}>
                  {page.views.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div style={{ marginTop: '16px' }}>
        <button
          style={{
            padding: '8px 16px',
            background: '#333',
            color: '#fff',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
          onClick={() => {
            (window as any).__dark_bridge.callServerTool('dashboard', { period });
          }}
        >
          Refresh Data
        </button>
      </div>
    </div>
  );
}

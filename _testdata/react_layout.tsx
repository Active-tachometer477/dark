import React from 'react';

export default function TestLayout({ children, title }: any) {
  return (
    <html>
      <head><title>{title || 'Test'}</title></head>
      <body>{children}</body>
    </html>
  );
}

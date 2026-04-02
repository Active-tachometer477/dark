import { h } from 'preact';

export default function Layout({ children }: any) {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>Dark App</title>
      </head>
      <body>
        {children}
      </body>
    </html>
  );
}

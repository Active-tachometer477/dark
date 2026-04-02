import { h } from 'preact';

export default function Index({ message }: any) {
  return (
    <div style="max-width: 600px; margin: 4rem auto; font-family: system-ui, sans-serif; text-align: center;">
      <h1>{message}</h1>
      <p style="color: #666;">Your dark app is running.</p>
    </div>
  );
}

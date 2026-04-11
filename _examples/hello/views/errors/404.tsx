export default function Error404({ path, statusCode }: any) {
  return (
    <div style="text-align: center; padding: 4rem 2rem;">
      <h1 style="font-size: 4rem; color: #e94560; margin-bottom: 0.5rem;">{statusCode}</h1>
      <p style="font-size: 1.2rem; color: #555;">Page Not Found</p>
      <p style="color: #888; font-family: monospace;">{path}</p>
      <a href="/" style="color: #3498db;">Back to Home</a>
    </div>
  );
}

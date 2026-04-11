export default function Error404({ path, statusCode }: any) {
  return (
    <div class="not-found-page">
      <h1>{statusCode} Not Found</h1>
      <p class="not-found-path">{path}</p>
    </div>
  );
}

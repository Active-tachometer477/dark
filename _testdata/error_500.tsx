import { h } from 'preact';

export default function Error500({ message, statusCode }: any) {
  return (
    <div class="error-page">
      <h1>Error {statusCode}</h1>
      <p class="error-message">{message}</p>
    </div>
  );
}

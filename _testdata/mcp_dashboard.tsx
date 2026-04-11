export default function Dashboard({ title, items }: any) {
  return (
    <div class="dashboard">
      <h1>{title || 'Dashboard'}</h1>
      <ul>
        {(items || []).map((item: any, i: number) => (
          <li key={i}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

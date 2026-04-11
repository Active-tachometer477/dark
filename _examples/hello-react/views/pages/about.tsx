export default function AboutPage({ title, features }: any) {
  return (
    <div>
      <h1>{title}</h1>
      <p>Dark is a Go SSR web framework. This example uses React instead of Preact.</p>
      <h2>Features</h2>
      <ul>
        {features?.map((f: string, i: number) => (
          <li key={i}>{f}</li>
        ))}
      </ul>
      <p>
        Switch between Preact and React with a single option:
      </p>
      <pre>{`dark.WithUILibrary(dark.React)`}</pre>
    </div>
  );
}

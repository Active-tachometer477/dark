export default function PageWithHead({ post, _head }: any) {
  return (
    <div>
      <dark-head>
        <title>{post?.title || 'Default Title'} | Blog</title>
        <meta name="description" content={post?.excerpt || 'No description'} />
        <meta property="og:title" content={post?.title || 'Default'} />
      </dark-head>
      <h1>{post?.title || 'No Post'}</h1>
      <p>{post?.body || ''}</p>
    </div>
  );
}

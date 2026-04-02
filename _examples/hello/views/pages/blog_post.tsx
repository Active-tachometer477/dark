import { h } from 'preact';
import './style.css';

export default function BlogPost({ post }: any) {
  return (
    <div class="blog-post">
      <dark-head>
        <title>{post.title} | Dark Blog</title>
        <meta name="description" content={post.excerpt} />
        <meta property="og:title" content={post.title} />
        <meta property="og:description" content={post.excerpt} />
      </dark-head>

      <h1>{post.title}</h1>
      <div class="blog-meta">{post.date} &middot; {post.author}</div>
      <div class="blog-body">
        {post.body.split('\n\n').map((p: string, i: number) => <p key={i}>{p}</p>)}
      </div>
      <p style="margin-top: 2rem;"><a href="/blog">&larr; Back to Blog</a></p>
    </div>
  );
}

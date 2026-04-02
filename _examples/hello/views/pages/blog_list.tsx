import { h } from 'preact';
import './style.css';

export default function BlogList({ posts }: any) {
  return (
    <div>
      <h1 style="margin-bottom: 1.5rem;">Blog</h1>
      <ul class="blog-list">
        {posts?.map((post: any) => (
          <li key={post.slug} class="blog-card">
            <h2><a href={`/blog/${post.slug}`}>{post.title}</a></h2>
            <div class="blog-meta">{post.date} &middot; {post.author}</div>
            <p class="blog-excerpt">{post.excerpt}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}

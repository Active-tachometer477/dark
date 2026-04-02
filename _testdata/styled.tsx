import { h } from 'preact';
import './style.css';

export default function StyledPage({ name }) {
  return <div class="test-styled">Styled {name}</div>;
}

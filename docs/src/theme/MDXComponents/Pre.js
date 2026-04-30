import React from 'react';
export default function MDXPre(props) {
  // With MDX 2, this element is only used for fenced code blocks
  // It always receives a MDXComponents/Code as children
  return <>{props.children}</>;
}

import React from 'react';
import CodeBlock from '@theme/CodeBlock';
export default function MDXCode(props) {
  const shouldBeInline = React.Children.toArray(props.children).every(
    (el) => typeof el === 'string' && !el.includes('\n'),
  );
  return shouldBeInline ? <code {...props} /> : <CodeBlock {...props} />;
}

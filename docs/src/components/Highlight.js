import React from 'react';

export default function Highlight({children}) {
  return (
    <span
      style={{
        borderRadius: '4px',
        backgroundColor: '#3F4C60',
        color: '#fff',
        padding: '0.2rem',
      }}>
      {children}
    </span>
  );
}

import React from 'react';

// Inline badges for marking individual sections of a page. Whole pages should
// instead be listed in enterprise-features.json, which badges both the page
// title and its sidebar entry.
export function Enterprise() {
  return <span className="tag enterprise"></span>;
}

export function Beta() {
  return <span className="tag beta"></span>;
}

export default function FeatureBadges({enterprise, beta}) {
  if (!enterprise && !beta) {
    return null;
  }
  return (
    <p>
      {enterprise && <Enterprise />}
      {enterprise && beta && ' '}
      {beta && <Beta />}
    </p>
  );
}

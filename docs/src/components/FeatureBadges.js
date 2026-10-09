import React from 'react';

// Inline badges for marking individual sections of a page. Whole pages should
// instead be listed in enterprise-features.json, which badges both the page
// title and its sidebar entry.
export function Pro() {
  return <span className="tag professional"></span>;
}

export function Beta() {
  return <span className="tag beta"></span>;
}

export default function FeatureBadges({pro, beta}) {
  if (!pro && !beta) {
    return null;
  }
  return (
    <p>
      {pro && <Pro />}
      {pro && beta && ' '}
      {beta && <Beta />}
    </p>
  );
}

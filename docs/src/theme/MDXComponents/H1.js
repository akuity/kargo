import React from 'react';
import {useDoc} from '@docusaurus/plugin-content-docs/client';
import MDXHeading from '@theme/MDXComponents/Heading';
import FeatureBadges from '@site/src/components/FeatureBadges';
import {isBeta, isEnterprise} from '@site/tags';

// Renders a page's title followed by any Enterprise/Beta badges listed for it in
// enterprise-features.json, so the same list drives both the sidebar and the
// page.
export default function MDXH1(props) {
  const {metadata} = useDoc();
  return (
    <>
      <MDXHeading as="h1" {...props} />
      <FeatureBadges
        enterprise={isEnterprise(metadata.id)}
        beta={isBeta(metadata.id)}
      />
    </>
  );
}

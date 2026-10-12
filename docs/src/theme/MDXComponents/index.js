import React from 'react';
import Head from '@docusaurus/Head';
import MDXCode from '@theme/MDXComponents/Code';
import MDXA from '@theme/MDXComponents/A';
import MDXPre from '@theme/MDXComponents/Pre';
import MDXDetails from '@theme/MDXComponents/Details';
import MDXHeading from '@theme/MDXComponents/Heading';
import MDXH1 from '@theme/MDXComponents/H1';
import MDXUl from '@theme/MDXComponents/Ul';
import MDXImg from '@theme/MDXComponents/Img';
import Admonition from '@theme/Admonition';
import Mermaid from '@theme/Mermaid';

// We use these a lot. Let's make them available globally.
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

// Custom highlight component
import Highlight from '@site/src/components/Highlight';

// Enterprise/Beta badges for individual sections of a page
import {Enterprise, Beta} from '@site/src/components/FeatureBadges';

const MDXComponents = {
  Head,
  details: MDXDetails,
  Details: MDXDetails,
  code: MDXCode,
  a: MDXA,
  pre: MDXPre,
  ul: MDXUl,
  img: MDXImg,
  h1: MDXH1,
  h2: (props) => <MDXHeading as="h2" {...props} />,
  h3: (props) => <MDXHeading as="h3" {...props} />,
  h4: (props) => <MDXHeading as="h4" {...props} />,
  h5: (props) => <MDXHeading as="h5" {...props} />,
  h6: (props) => <MDXHeading as="h6" {...props} />,
  admonition: Admonition,
  mermaid: Mermaid,

  // We use these a lot. Let's make them available globally.
  Tabs,
  TabItem,

  // Custom highlight component
  Hlt: Highlight,

  // Enterprise/Beta badges for individual sections of a page
  Enterprise,
  Beta,
};

export default MDXComponents;

import React from 'react';
import Head from '@docusaurus/Head';

type Props = {
  version: string;
  datePublished: string;
};

export default function ReleaseNotesSchema({version, datePublished}: Props) {
  const schema = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: `Kargo ${version} Release Notes`,
    datePublished,
    dateModified: datePublished,
    url: `https://docs.kargo.io/release-notes/${version}`,
    inLanguage: 'en',
    isPartOf: {
      '@type': 'TechArticle',
      name: 'Kargo Release Notes',
    },
    author: {
      '@type': 'Organization',
      name: 'Akuity',
      url: 'https://akuity.io',
    },
    publisher: {
      '@type': 'Organization',
      name: 'Akuity',
      url: 'https://akuity.io',
    },
    about: {
      '@type': 'SoftwareApplication',
      name: 'Kargo',
      softwareVersion: version.replace(/^v/, ''),
    },
  };
  return (
    <Head>
      <script type="application/ld+json">{JSON.stringify(schema)}</script>
    </Head>
  );
}

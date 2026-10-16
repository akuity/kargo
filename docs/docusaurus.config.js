// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const path = require('node:path');
const {themes} = require('prism-react-renderer');
const tags = require('./tags');
const enterpriseFeatures = require('./enterprise-features.json');
const promotionStepGroups = require('./promotion-step-groups.json');
const lightCodeTheme = themes.github;
const darkCodeTheme = themes.dracula;

/** @type {string} */
const siteDescription = 'Kargo is a Kubernetes-native continuous promotion platform that orchestrates the movement of artifacts through environments in GitOps workflows.';

const softwareApplicationJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'Kargo',
  description: siteDescription,
  applicationCategory: 'DeveloperApplication',
  operatingSystem: 'Kubernetes',
  codeRepository: 'https://github.com/akuity/kargo',
  license: 'https://www.apache.org/licenses/LICENSE-2.0',
  url: 'https://kargo.io',
  publisher: {
    '@type': 'Organization',
    name: 'Akuity',
    url: 'https://akuity.io',
  },
  offers: {
    '@type': 'Offer',
    price: '0',
    priceCurrency: 'USD',
  },
};

const config = {
  title: 'Kargo Docs',
  tagline: siteDescription,
  url: 'https://docs.kargo.io',
  baseUrl: '/',
  trailingSlash: false,
  onBrokenLinks: 'throw',
  onBrokenAnchors: 'throw',
  favicon: 'img/kargo.png',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },
  headTags: [
    {
      tagName: 'script',
      attributes: {type: 'application/ld+json'},
      innerHTML: JSON.stringify(softwareApplicationJsonLd),
    },
  ],

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          sidebarCollapsible: false,
          routeBasePath: '/', // Serve the docs at the site's roo
          sidebarItemsGenerator: async function ({
            defaultSidebarItemsGenerator,
            ...args
          }) {
            // Fail fast on typos or moved/deleted docs, which would otherwise
            // silently drop their badges.
            const docIds = new Set(args.docs.map((doc) => doc.id));
            const listed = new Set([
              ...enterpriseFeatures.enterprise,
              ...enterpriseFeatures.beta,
            ]);
            const unknown = [...listed].filter((id) => !docIds.has(id));
            if (unknown.length) {
              throw new Error(`enterprise-features.json lists unknown doc IDs: ${unknown.join(', ')}`);
            }

            const sidebarItems = await defaultSidebarItemsGenerator(args);

            // Class Enterprise features (per enterprise-features.json) so
            // custom.css can badge them. Categories are matched by the doc
            // they link to (i.e. their index page).
            function addBadges(items) {
              return items.map((item) => {
                if (item.type === 'category') {
                  item.items = addBadges(item.items);
                }
                const docId = item.type === 'doc' ? item.id : item.link?.id;
                if (docId && tags.isEnterprise(docId)) {
                  item.className = [item.className, 'sidebar-enterprise']
                    .filter(Boolean)
                    .join(' ');
                }
                return item;
              });
            }
            // Arrange promotion steps into the groups in
            // promotion-step-groups.json. custom.css hides them from the
            // sidebar; they drive the catalog page, breadcrumbs, and
            // previous/next order. Every step must be listed in exactly one
            // group so new steps can't silently go missing.
            const stepsDir = 'user-guide/reference-docs/promotion-steps/';
            const stepsIndex = `${stepsDir}index`;
            const groupedStepIds = new Set();
            for (const group of promotionStepGroups) {
              for (const step of group.steps) {
                const docId = stepsDir + step;
                if (!docIds.has(docId)) {
                  throw new Error(`promotion-step-groups.json lists unknown step: ${step}`);
                }
                if (groupedStepIds.has(docId)) {
                  throw new Error(`promotion-step-groups.json lists step more than once: ${step}`);
                }
                groupedStepIds.add(docId);
              }
            }
            const missingSteps = [...docIds]
              .filter((id) => (
                id.startsWith(stepsDir) &&
                id !== stepsIndex &&
                !groupedStepIds.has(id)
              ))
              .map((id) => id.slice(stepsDir.length));
            if (missingSteps.length) {
              throw new Error(`promotion-step-groups.json is missing steps: ${missingSteps.join(', ')}`);
            }

            function groupSteps(items) {
              return items.map((item) => {
                if (item.type !== 'category') {
                  return item;
                }
                if (item.link?.id !== stepsIndex) {
                  item.items = groupSteps(item.items);
                  return item;
                }
                // Groups and the steps within them appear in the order they're
                // listed in promotion-step-groups.json.
                const stepsById = new Map(
                  item.items.map((step) => [step.id, step]),
                );
                item.className = 'sidebar-step-catalog';
                item.items = promotionStepGroups.map((group) => ({
                  type: 'category',
                  label: group.label,
                  items: group.steps.map((step) => stepsById.get(stepsDir + step)),
                }));
                return item;
              });
            }

            return groupSteps(addBadges(sidebarItems));
          },
        },
        blog: false,
        pages: {},
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  plugins: [
    [
      path.join(__dirname, "plugins", "gtag", "lib"),
      {
        trackingID: 'G-0LYKG06H98',
        anonymizeIP: true,
      },
    ],
    'docusaurus-plugin-sass',
    [
      '@scalar/docusaurus',
      {
        label: 'API Reference',
        route: '/api-docs',
        showNavLink: false, // Don't include in the top navbar
        configuration: {
          url: '/swagger.json',
          servers: [{
            url: '{baseUrl}',
            variables: {
              baseUrl: {default: 'https://kargo.example.com'}
            }
          }]
        },
      }
    ],
    [
      'docusaurus-plugin-llms',
      {
        // The curated llms.txt in static/ is the canonical index; only the
        // full-content bundle is auto-generated.
        generateLLMsTxt: false,
        generateLLMsFullTxt: true,
        docsDir: 'docs',
        title: 'Kargo Documentation',
        description: 'Kargo is a Kubernetes-native continuous promotion platform for GitOps workflows.',
        excludeImports: true,
        removeDuplicateHeadings: true,
      },
    ],
  ],
  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      docs: {
        sidebar: {
          hideable: true,
        },
      },
      navbar: {
        title: 'Kargo Docs',
        logo: {
          alt: 'Kargo Documentation',
          src: 'img/kargo.png',
          href: '/',
          target: '_self',
        },
        items: [
          {
            href: 'https://akuity.io/',
            label: 'Akuity.io',
            position: 'left',
          },
          {
            type: 'custom-version-dropdown',
            position: 'right',
          },
          {
            href: 'https://kargo.io/',
            label: 'Kargo.io',
            position: 'left',
          },
          {
            href: 'https://github.com/akuity/kargo',
            label: 'GitHub',
            position: 'right',
          },
          {
            href: 'http://akuity.community',
            label: 'Discord Community',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        copyright: `Copyright © ${new Date().getFullYear()} Akuity`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
      colorMode: {
        defaultMode: 'light',
      },
      metadata: [
        {
          name: 'keywords',
          content: 'kargo, akuity, argoproj, argo cd, argo workflows, argo events, argo rollouts, kubernetes, gitops, devops, continuous promotion',
        },
        {name: 'description', content: siteDescription},
      ],
      algolia: {
        appId: '3SQ7LK6WD9',
        apiKey: '5627b8c2efd5b28a5b70c6660cb2b0f3',
        indexName: 'kargo',
        contextualSearch: true,
      }
    }),
};

module.exports = config;

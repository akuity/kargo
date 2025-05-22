import React, { ReactNode, useEffect } from 'react';
import clsx from 'clsx';
import { FaRocket, FaTruck, FaExternalLinkAlt } from 'react-icons/fa';
import Link from '@docusaurus/Link';
import styles from './index.module.scss';

export default function Home(): JSX.Element {
  useEffect(() => {
    const breadcrumb = document.querySelector('[aria-label="Breadcrumbs"]') as HTMLElement;

    if (breadcrumb) {
      breadcrumb.style.display = 'none';
    }

    return () => {
      document.body.style.background = '';

      if (breadcrumb) {
        breadcrumb.style.display = '';
      }
    }
  }, []);

  return (
    <main className={styles.landingPage}>
      <div className={styles.hero}>
        <div className="container">
          <div className={styles.heroContent}>
            <div className={styles.heroImage}>
              <img
                src="/img/3d-mascotte.png"
                alt="Kargo GitOps Mascot"
                className={styles.heroImg}
              />
            </div>
            <div className={styles.heroText}>
              <h1 className={styles.heroTitle}>Kargo</h1>
              <p className={styles.heroSubtitle}>
                Learn how to use Kargo for GitOps promotions
              </p>
              <div className={styles.heroActions}>
                <Link to='/user-guide/core-concepts' className={styles.heroButton}>
                  Overview
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="container">
        <Section
          title={<><FaRocket className={styles.sectionIcon} /> Get Started</>}
          description="Everything you need to begin your journey with Kargo"
          cards={[
            {
              id: 'install',
              title: 'Installation',
              description: 'Set up Kargo in your environment with our comprehensive installation guides.',
              links: [
                { label: 'Install with Helm', to: '/operator-guide/basic-installation' },
                { label: 'Install with ArgoCD', to: '/operator-guide/advanced-installation/advanced-with-argocd' }
              ],
              color: '#1CAC77',
              href: '/operator-guide/'
            },
            {
              id: 'users-guide',
              title: 'User\'s Guide',
              description: 'Learn key concepts and fundamentals of Kargo to get productive quickly.',
              color: '#FE7537',
              href: '/user-guide/'
            },
            {
              id: 'quickstart',
              title: 'Quickstart',
              description: 'Basic introduction to Kargo in your Kubernetes cluster with hands-on examples.',
              color: '#1DCECA',
              href: '/quickstart/'
            },
            {
              id: 'examples',
              title: 'Examples',
              description: 'Learn Kargo through practical examples and real-world use cases.',
              color: '#E85A4F',
              href: '/user-guide/examples/'
            },
          ]}
        />

        <Section
          title={<><FaTruck className={styles.sectionIcon} /> Advance in Kargo</>}
          description="Dive deeper into advanced features and join our community"
          cards={[
            {
              id: 'ref-docs',
              title: 'References',
              description: 'Comprehensive reference documentation for all Kargo components and features.',
              links: [
                { label: 'CRD Documentation', to: 'https://doc.crds.dev/github.com/akuity/kargo', external: true },
                { label: 'Promotion Steps', to: '/user-guide/reference-docs/promotion-steps' },
                { label: 'Expression Language', to: '/user-guide/reference-docs/expressions' }
              ],
              color: '#f1619b'
            },
            {
              id: 'community',
              title: 'Join the Community',
              description: 'Ask questions, learn from others, and improve together in the Akuity Discord community.',
              color: '#A9499D',
              href: 'https://akuity.community',
              external: true
            },
            {
              id: 'contribute',
              title: 'Contribute',
              description: 'Help make Kargo better by contributing code, documentation, or feedback.',
              color: '#6380E1',
              href: '/contributor-guide/'
            }
          ]}
        />
      </div>
    </main>
  );
}

type CardLink = {
  label: string;
  to: string;
  external?: boolean;
};

type CardProps = {
  id: string;
  title: string;
  description: string;
  color: string;
  href?: string;
  external?: boolean;
  links?: CardLink[];
};

const Card = ({ title, description, color, href, external, links }: CardProps) => {
  const cardContent = (
    <div className={styles.card}>
      <div
        className={styles.cardHeader}
        style={{ backgroundColor: color }}
      >
        <h3 className={styles.cardTitle}>{title}</h3>
      </div>
      <div className={styles.cardBody}>
        <p className={styles.cardDescription}>{description}</p>
        {links && links.length > 0 && (
          <div className={styles.cardLinks}>
            {links.map((link, index) => (
              <Link
                key={index}
                to={link.to}
                className={styles.cardLink}
                {...(link.external && { target: '_blank', rel: 'noopener noreferrer' })}
              >
                {link.label}
                {link.external && <FaExternalLinkAlt className={styles.externalIcon} />}
              </Link>
            ))}
          </div>
        )}
      </div>
      {href && (
        <div className={styles.cardFooter}>
          <span className={styles.cardCta}>
            Learn more →
          </span>
        </div>
      )}
    </div>
  );

  const wrapperStyle = { '--card-hover-color': color } as React.CSSProperties;

  if (href) {
    return (
      <Link
        to={href}
        className={styles.cardWrapper}
        style={wrapperStyle}
        {...(external && { target: '_blank', rel: 'noopener noreferrer' })}
      >
        {cardContent}
      </Link>
    );
  }

  return (
    <div className={styles.cardWrapper} style={wrapperStyle}>
      {cardContent}
    </div>
  );
};

type SectionProps = {
  title: ReactNode;
  description: string;
  cards: CardProps[];
};

const Section = ({ title, description, cards }: SectionProps) => (
  <section className={styles.section}>
    <div className={styles.sectionHeader}>
      <h2 className={styles.sectionTitle}>{title}</h2>
      <p className={styles.sectionDescription}>{description}</p>
    </div>
    <div className={styles.cardGrid}>
      {cards.map((card) => (
        <Card key={card.id} {...card} />
      ))}
    </div>
  </section>
);

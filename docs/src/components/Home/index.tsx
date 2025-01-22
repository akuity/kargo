import React, { CSSProperties, ReactNode, useEffect } from 'react';

import styles from './index.module.scss';
import clsx from 'clsx';
import { FaRocket, FaTruck, FaTools } from 'react-icons/fa';
import Link, { NavLinkProps } from '@docusaurus/Link';
import { Button } from '../Button';

export default function Home(): JSX.Element {
  useEffect(() => {
    const breadcrumb = document.querySelector('[aria-label="Breadcrumbs"]') as HTMLElement;

    if (breadcrumb) {
      breadcrumb.style.display = 'none';
    }

    // document.body.style.background = 'linear-gradient(90deg,var(--generic-bg) calc(25px - 1px),transparent 1%) 50% / 25px 25px,linear-gradient(var(--generic-bg) calc(25px - 1px),transparent 1%) 50% /25px 25px,var(--generic-color)';
    return () => {
      document.body.style.background = '';

      if (breadcrumb) {
        breadcrumb.style.display = '';
      }
    }
  }, []);

  return (
      <main className='landing-page'>
        <header className={styles.header}>
          <h1 style={{margin: 0}}>Kargo</h1>
          <span>Learn how to use Kargo for GitOps promotions of stages</span>
          <div style={{marginTop: '24px'}}>
            <Link to='/new-docs/user-guide/core-concepts'>
              <Button>Overview</Button>
            </Link>
            <Link to='/new-docs/user-guide/examples'>
              <Button btnType='primary' style={{marginLeft: '24px'}}>Learn By Examples</Button>
            </Link>
            <div style={{
              marginTop: '24px',
              padding: '16px',
              border: '1px solid #ffcc00',
              backgroundColor: '#fff8e1',
              borderRadius: '4px',
              color: '#ff6f00'
            }}>
              <p>
                <FaTools style={{ marginRight: '8px' }} />
                <strong>We are actively re-writing and re-organizing this documentation.</strong>
              </p>
              <p>
                During this transition, both the old and new, work-in-progress documentation are available.
              </p>
            </div>
          </div>
        </header>
        <div className='container'>
          <Section 
            title={<><FaRocket /> Get Started</>}
            nodes={[
              {
                id: 'install',
                title: 'Installation',
                description: (
                  <>
                    <Link to='/new-docs/operator-guide/basic-installation' className='highlight'>Basic Installation</Link>
                    <br />
                    <Link to='/new-docs/operator-guide/advanced-installation/advanced-with-helm' className='highlight'>With Helm</Link>
                    <br />
                    <Link to='/new-docs/operator-guide/advanced-installation/advanced-with-argocd' className='highlight'>With ArgoCD</Link>
                  </>
                ),
                headerStyle: {
                  background: '#1CAC77',
                  color: 'white'
                },
                link: {
                  to: '/new-docs/operator-guide/'
                }
              },
              {
                id: 'users-guide',
                title: 'User\'s Guide',
                description: 'Learn key concepts',
                headerStyle: {
                  background: '#FE7537',
                  color: 'white'
                },
                link: {
                  to: '/new-docs/user-guide/'
                }
              },
              {
                id: 'quickstart',
                title: 'Quickstart',
                description: 'Basic introduction to kargo in your kubernetes cluster',
                headerStyle: {
                  background: '#1DCECA',
                  color: 'white'
                },
                link: {
                  to: '/new-docs/quickstart/'
                }
              },
            ]}
          />

          <Section 
            title={<><FaTruck /> Advance in Kargo</>}
            nodes={[
              {
                id: 'ref-docs',
                title: 'References',
                description: (
                  <>
                    <Link to='/new-docs/user-guide/reference-docs/crds' className='highlight'>CRD Docs</Link>
                    <br />
                    <Link to='/new-docs/user-guide/reference-docs/promotion-steps' className='highlight'>Promotion Steps</Link>
                    <br />
                    <Link to='/new-docs/user-guide/reference-docs/expressions' className='highlight'>Expression Language</Link>
                  </>
                ),
                headerStyle: {
                  background: '#f1619b',
                  color: 'white'
                },
              },
              {
                id: 'community',
                title: 'Join Community',
                description: 'Ask, learn and improve in Akuity Discord community',
                headerStyle: {
                  background: '#A9499D',
                  color: 'white'
                },
                link: {
                  to: 'https://akuity.community',
                  target: '_blank'
                }
              },
              {
                id: 'contribute',
                title: 'Contribute',
                description: 'Want to contribute to Kargo? Read Contributer\'s guide',
                headerStyle: {
                  background: '#6380E1',
                  color: 'white'
                },
                link: {
                  to: '/new-docs/contributor-guide/'
                }
              }
            ]}
          />
        </div>
      </main>
  );
}

type NodeProps = {
  id: string;
  title: string;
  description: ReactNode;
  headerStyle?: CSSProperties;
  link?: Pick<NavLinkProps, 'to' | 'target'>;
};

const Node = (props: NodeProps) => (
  <Link to={props.link?.to as string} target={props.link?.target}>
    <li className={styles.node} >
      <h4 className={styles.nodeHeader} style={props.headerStyle}>{props.title}</h4>
      <p className={styles.nodeBody}>{props.description}</p>
    </li>
  </Link>
);

type SectionProps = {
  title: ReactNode;
  nodes: NodeProps[];
};

const Section = (props: SectionProps) => (
  <section className={styles.section}>
    <h2 className={clsx(styles.title)}>{props.title}</h2>

    <ul className={styles.stages}>
      {props.nodes.map((node) => <Node key={node.id} {...node} />)}
    </ul>
  </section>
)

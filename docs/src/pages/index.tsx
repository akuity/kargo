import React, { CSSProperties, ReactNode, useEffect } from 'react';
import Layout from '@theme/Layout';

import styles from './index.module.css';
import clsx from 'clsx';
import { FaRocket, FaTruck } from 'react-icons/fa';
import Link, { NavLinkProps } from '@docusaurus/Link';

export default function Home(): JSX.Element {
  useEffect(() => {
    document.body.style.background = 'linear-gradient(90deg,#fff calc(25px - 1px),transparent 1%) 50% / 25px 25px,linear-gradient(#fff calc(25px - 1px),transparent 1%) 50% /25px 25px,#333';
    return () => {
      document.body.style.background = '';
    }
  }, []);

  return (
    <Layout
      // TODO: change title and description?
      title={`Kargo - GitOps Promotion Tool`}
      description="Do stage to stage GitOps promotion right way using Kargo">
      <main className='landing-page'>
        <header className={styles.header}>
          <h1 style={{margin: 0}}>Kargo</h1>
          <span>Learn how to use Kargo for GitOps promotions of stages</span>
          <div style={{marginTop: '24px'}}>
            <Link to='/new-docs/user-guide/core-concepts'>
              <button className='btn'>Overview</button>
            </Link>
            <Link to='/new-docs/user-guide/examples'>
              <button className='btn btn-primary' style={{marginLeft: '24px'}}>Learn By Examples</button>
            </Link>
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
                description: 'Get your hands dirty with kargo by following quickstart',
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
                id: 'crd',
                title: 'CRD Documentation',
                description: 'Read full CRD documentation',
                headerStyle: {
                  background: '#f1619b',
                  color: 'white'
                },
                link: {
                  to: '/new-docs/user-guide/reference-docs/crds'
                }
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
                description: 'Want to contribute to Kargo?',
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
    </Layout>
  );
}

type NodeProps = {
  id: string;
  title: string;
  description: ReactNode;
  headerStyle?: CSSProperties;
  link: Pick<NavLinkProps, 'to' | 'target'>;
};

const Node = (props: NodeProps) => (
  <Link to={props.link.to as string} target={props.link.target}>
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

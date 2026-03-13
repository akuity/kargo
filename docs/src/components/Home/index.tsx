import React, { ReactNode, useEffect, useState } from 'react';
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

  /* Modal state and escape key handler for zooming in on the Kargo UI screenshot. */
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const handleEsc = (e) => {
      if (e.key === "Escape") setIsOpen(false);
    };
    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, []);

  return (
    <main className={styles.landingPage}>
      <p className={styles.eyebrow}>
        From the creators of <strong>Argo CD</strong>
      </p>

      <div style={{ display: "flex", alignItems: "center", gap: "0rem" }}>
        <div>
          <h1 className={styles.heroTitle}>Kargo</h1>
          <h4>GitOps-native application promotion across environments</h4>
          <p>
            Kargo is an open-source project that makes application promotion across environments repeatable, auditable, and observable — turning release orchestration into a declarative, GitOps-native process.
          </p>
        </div>
        <img
          src="/img/3d-mascotte.png"
          alt="Kargo mascot"
          style={{ height: "200px", width: "auto", flexShrink: 0, marginLeft: "-1rem" }}
        />
      </div>

      <h1>Where Kargo Fits in Your Pipeline</h1>

      <p>
        Kargo sits between CI and CD and turns application promotion into a <strong>first-class, declarative process</strong>.
      </p>
      <p>
        Instead of embedding release logic in CI jobs or custom scripts, Kargo models promotion as state — making every step <strong>visible, versioned, and auditable</strong>.
      </p>

      <img
        src="/img/where-kargo-fits.svg"
        alt="Where Kargo fits in the CI/CD pipeline"
      />

      <h1>How It Works</h1>

      <p>
        Kargo models promotion as code:

        <ul className={styles.bulletList}>
          <li><strong>Stages</strong> define your environments (dev, staging, prod) <i style={{color:'grey'}}>or promotion targets</i></li>
          <li><strong>Freight</strong> represents versioned application artifacts</li>
          <li><strong>Promotion</strong> policies control how and when versions move forward</li>
          <li>Every promotion is <strong>visible, versioned, and auditable</strong></li>  
        </ul>
      </p>

      <p>
      </p>

      <img
        src="/img/concepts.svg"
        alt="Kargo core concepts diagram"
      />

      <h1>Without Kargo</h1>

      <p>
        Promotion logic lives inside CI jobs and scripts.

        <ul className={styles.bulletList}>
          <li>Promotion is tightly coupled to build pipelines</li>
          <li>Behavior is spread across pipeline steps and custom automation</li>
          <li>Failures are hard to trace across environments</li>
          <li>Rollbacks require manual intervention</li>
          <li>Audit trails are fragmented</li>
        </ul>
      </p>

      <div className={styles.beforeKargo}>
        <header>
          <div className={styles.legend}>
            <div className={styles.legendItem}><div className={styles.legendDot} style={{background:'#0d7a4a'}}></div>Automated step</div>
            <div className={styles.legendItem}><div className={styles.legendDot} style={{background:'#a06400'}}></div>Human approval gate</div>
            <div className={styles.legendItem}><div className={styles.legendDot} style={{background:'#1a5fc9'}}></div>GitOps sync (Argo CD)</div>
            <div className={styles.legendItem}><div className={styles.legendDot} style={{background:'#cc2a12'}}></div>Validation / smoke test</div>
          </div>
        </header>

        <div className={styles.diagramWrapper}>
          <svg className={styles.seqSvg} viewBox="0 0 860 940" xmlns="http://www.w3.org/2000/svg" style={{maxWidth:'100%'}}>
            <defs>
              <marker id="arr-auto" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
                <path d="M0,0 L0,6 L8,3 z" fill="#0d7a4a"/>
              </marker>
              <marker id="arr-gate" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
                <path d="M0,0 L0,6 L8,3 z" fill="#a06400"/>
              </marker>
              <marker id="arr-sync" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
                <path d="M0,0 L0,6 L8,3 z" fill="#1a5fc9"/>
              </marker>
              <marker id="arr-validate" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
                <path d="M0,0 L0,6 L8,3 z" fill="#cc2a12"/>
              </marker>
              <marker id="arr-notify" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
                <path d="M0,0 L0,6 L8,3 z" fill="#7a82a8"/>
              </marker>
              <marker id="arr-white" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
                <path d="M0,0 L0,6 L8,3 z" fill="#2c3152"/>
              </marker>
            </defs>

            <rect width="860" height="940" fill="#ffffff"/>

            {/* Phase background bands */}
            <rect x="20" y="60" width="820" height="185" rx="6" fill="#f2fbf6" stroke="#a8dfc0" strokeWidth="1"/>
            <text x="36" y="82" fontFamily="JetBrains Mono" fontSize="11" fontWeight="700" fill="#0d7a4a" letterSpacing="2">DEV</text>

            <rect x="20" y="260" width="820" height="200" rx="6" fill="#f0f5ff" stroke="#a8c4ee" strokeWidth="1"/>
            <text x="36" y="282" fontFamily="JetBrains Mono" fontSize="11" fontWeight="700" fill="#1a5fc9" letterSpacing="2">STAGING</text>

            <rect x="20" y="485" width="820" height="215" rx="6" fill="#fffbf0" stroke="#e8d080" strokeWidth="1"/>
            <text x="36" y="497" fontFamily="JetBrains Mono" fontSize="11" fontWeight="700" fill="#a06400" letterSpacing="2">PROD · US-EAST</text>

            <rect x="20" y="705" width="820" height="215" rx="6" fill="#fffbf0" stroke="#e8d080" strokeWidth="1"/>
            <text x="36" y="727" fontFamily="JetBrains Mono" fontSize="11" fontWeight="700" fill="#a06400" letterSpacing="2">PROD · US-WEST</text>

            {/* Actor headers */}
            <rect x="58" y="8" width="84" height="44" rx="8" fill="#f5f6fa" stroke="#dde1ee" strokeWidth="1"/>
            <text x="100" y="30" fontFamily="JetBrains Mono" fontSize="18" textAnchor="middle" fill="#2c3152">👤</text>
            <text x="100" y="44" fontFamily="JetBrains Mono" fontSize="8" fontWeight="600" fill="#7a82a8" textAnchor="middle" letterSpacing="1">DEVELOPER</text>

            <rect x="228" y="8" width="84" height="44" rx="8" fill="#f5f6fa" stroke="#dde1ee" strokeWidth="1"/>
            <g transform="translate(270 30)">
              <image href="https://upload.wikimedia.org/wikipedia/commons/e/e9/Jenkins_logo.svg" x="-14" y="-18" width="30" height="30"/>
            </g>
            <text x="270" y="50" fontFamily="JetBrains Mono" fontSize="8" fontWeight="600" fill="#7a82a8" textAnchor="middle" letterSpacing="1">CI PIPELINE</text>

            <rect x="418" y="8" width="84" height="44" rx="8" fill="#f5f6fa" stroke="#dde1ee" strokeWidth="1"/>
            <g transform="translate(460 26)">
              <image href="https://git-scm.com/images/logos/downloads/Git-Icon-1788C.svg" x="-14" y="-13" width="28" height="28"/>
            </g>
            <text x="460" y="48" fontFamily="JetBrains Mono" fontSize="8" fontWeight="600" fill="#7a82a8" textAnchor="middle" letterSpacing="1">GIT REPO</text>

            <rect x="618" y="8" width="84" height="44" rx="8" fill="#f5f6fa" stroke="#dde1ee" strokeWidth="1"/>
            <text x="660" y="30" fontFamily="JetBrains Mono" fontSize="18" textAnchor="middle" fill="#2c3152">🐙</text>
            <text x="660" y="44" fontFamily="JetBrains Mono" fontSize="8" fontWeight="600" fill="#7a82a8" textAnchor="middle" letterSpacing="1">ARGO CD</text>

            {/* Lifelines */}
            <line x1="100" y1="52" x2="100" y2="940" stroke="#c4c9de" strokeWidth="1.5" strokeDasharray="6 4"/>
            <line x1="270" y1="52" x2="270" y2="940" stroke="#c4c9de" strokeWidth="1.5" strokeDasharray="6 4"/>
            <line x1="460" y1="52" x2="460" y2="940" stroke="#c4c9de" strokeWidth="1.5" strokeDasharray="6 4"/>
            <line x1="660" y1="52" x2="660" y2="940" stroke="#c4c9de" strokeWidth="1.5" strokeDasharray="6 4"/>

            {/* DEV PHASE */}
            <line x1="660" y1="100" x2="660" y2="115" stroke="#dde1ee" strokeWidth="1.5"/>
            <rect x="648" y="100" width="24" height="28" rx="4" fill="#1a5fc9" opacity="0.9"/>
            <text x="676" y="110" fontFamily="JetBrains Mono" fontSize="11" fill="#1a5fc9" fontWeight="500">sync dev</text>
            <path d="M660,115 Q720,115 720,124 Q720,133 660,133" stroke="#1a5fc9" strokeWidth="1.5" fill="none" markerEnd="url(#arr-sync)"/>

            <line x1="660" y1="142" x2="270" y2="142" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)" strokeDasharray="4 3"/>
            <text x="430" y="138" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">webhook → trigger build</text>

            <rect x="258" y="148" width="24" height="60" rx="4" fill="#cc2a12" opacity="0.85"/>
            <text x="252" y="186" fontFamily="JetBrains Mono" fontSize="11" fill="#2c3152" textAnchor="end">validate dev</text>

            <line x1="282" y1="188" x2="460" y2="188" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)"/>
            <text x="370" y="183" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">update staging manifest</text>

            {/* STAGING PHASE */}
            <rect x="648" y="278" width="24" height="28" rx="4" fill="#1a5fc9" opacity="0.9"/>
            <text x="676" y="288" fontFamily="JetBrains Mono" fontSize="11" fill="#1a5fc9" fontWeight="500">sync staging</text>
            <path d="M660,293 Q724,293 724,303 Q724,312 660,312" stroke="#1a5fc9" strokeWidth="1.5" fill="none" markerEnd="url(#arr-sync)"/>

            <line x1="660" y1="320" x2="270" y2="320" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)" strokeDasharray="4 3"/>
            <text x="430" y="315" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">webhook → trigger tests</text>

            <rect x="258" y="326" width="24" height="50" rx="4" fill="#cc2a12" opacity="0.85"/>
            <text x="252" y="352" fontFamily="JetBrains Mono" fontSize="11" fill="#2c3152" textAnchor="end">validate staging</text>

            <line x1="258" y1="386" x2="100" y2="386" stroke="#a06400" strokeWidth="1.5" markerEnd="url(#arr-gate)"/>
            <text x="175" y="380" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle">notify — awaiting approval</text>
            <rect x="160" y="389" width="30" height="12" rx="3" fill="#a06400" opacity="0.15" stroke="#a06400" strokeWidth="0.5"/>
            <text x="175" y="399" fontFamily="JetBrains Mono" fontSize="7" fill="#a06400" textAnchor="middle">🔒 GATE</text>

            <rect x="90" y="386" width="20" height="136" rx="3" fill="#a06400" opacity="0.25"/>

            {/* PROD US-EAST */}
            <line x1="110" y1="520" x2="258" y2="520" stroke="#a06400" strokeWidth="2" markerEnd="url(#arr-gate)"/>
            <text x="175" y="514" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle">approval (us-east)</text>

            <line x1="282" y1="528" x2="460" y2="528" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)"/>
            <text x="370" y="522" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">update prod/us-east manifest</text>

            <rect x="648" y="545" width="24" height="28" rx="4" fill="#1a5fc9" opacity="0.9"/>
            <text x="676" y="555" fontFamily="JetBrains Mono" fontSize="11" fill="#1a5fc9" fontWeight="500">sync prod (us-east)</text>
            <path d="M660,560 Q730,560 730,569 Q730,578 660,578" stroke="#1a5fc9" strokeWidth="1.5" fill="none" markerEnd="url(#arr-sync)"/>

            <line x1="660" y1="586" x2="270" y2="586" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)" strokeDasharray="4 3"/>
            <text x="430" y="581" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">webhook → run smoke tests</text>

            <rect x="258" y="592" width="24" height="50" rx="4" fill="#cc2a12" opacity="0.85"/>
            <text x="252" y="614" fontFamily="JetBrains Mono" fontSize="11" fill="#2c3152" textAnchor="end">validate prod (us-east)</text>

            <line x1="258" y1="648" x2="100" y2="648" stroke="#a06400" strokeWidth="1.5" markerEnd="url(#arr-gate)"/>
            <text x="175" y="642" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle">notify — awaiting approval</text>
            <rect x="160" y="651" width="30" height="12" rx="3" fill="#a06400" opacity="0.15" stroke="#a06400" strokeWidth="0.5"/>
            <text x="175" y="661" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle">🔒 GATE</text>

            <rect x="90" y="648" width="20" height="102" rx="3" fill="#a06400" opacity="0.25"/>

            {/* PROD US-WEST */}
            <line x1="110" y1="750" x2="258" y2="750" stroke="#a06400" strokeWidth="2" markerEnd="url(#arr-gate)"/>
            <text x="175" y="744" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle">approval (us-west)</text>

            <line x1="282" y1="758" x2="460" y2="758" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)"/>
            <text x="370" y="752" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">update prod/us-west manifest</text>

            <rect x="648" y="773" width="24" height="28" rx="4" fill="#1a5fc9" opacity="0.9"/>
            <text x="676" y="781" fontFamily="JetBrains Mono" fontSize="11" fill="#1a5fc9" fontWeight="500">sync prod (us-west)</text>
            <path d="M660,788 Q730,788 730,797 Q730,806 660,806" stroke="#1a5fc9" strokeWidth="1.5" fill="none" markerEnd="url(#arr-sync)"/>

            <line x1="660" y1="815" x2="270" y2="815" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)" strokeDasharray="4 3"/>
            <text x="430" y="810" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">webhook → run smoke tests</text>

            <rect x="258" y="820" width="24" height="50" rx="4" fill="#cc2a12" opacity="0.85"/>
            <text x="252" y="842" fontFamily="JetBrains Mono" fontSize="11" fill="#2c3152" textAnchor="end">validate prod (us-west)</text>

            <line x1="258" y1="878" x2="100" y2="878" stroke="#0d7a4a" strokeWidth="1.5" markerEnd="url(#arr-auto)"/>
            <text x="175" y="872" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle">✓ deploy complete</text>

            {/* Step numbers */}
            <circle cx="40" cy="142" r="9" fill="#f5f6fa" stroke="#0d7a4a" strokeWidth="1"/>
            <text x="40" y="146" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle" fontWeight="700">1</text>

            <circle cx="40" cy="188" r="9" fill="#f5f6fa" stroke="#0d7a4a" strokeWidth="1"/>
            <text x="40" y="192" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle" fontWeight="700">2</text>

            <circle cx="40" cy="320" r="9" fill="#f5f6fa" stroke="#1a5fc9" strokeWidth="1"/>
            <text x="40" y="324" fontFamily="JetBrains Mono" fontSize="11" fill="#1a5fc9" textAnchor="middle" fontWeight="700">3</text>

            <circle cx="40" cy="386" r="9" fill="#f5f6fa" stroke="#a06400" strokeWidth="1"/>
            <text x="40" y="390" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle" fontWeight="700">4</text>

            <circle cx="40" cy="510" r="9" fill="#f5f6fa" stroke="#a06400" strokeWidth="1"/>
            <text x="40" y="514" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle" fontWeight="700">5</text>

            <circle cx="40" cy="648" r="9" fill="#f5f6fa" stroke="#a06400" strokeWidth="1"/>
            <text x="40" y="652" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle" fontWeight="700">6</text>

            <circle cx="40" cy="740" r="9" fill="#f5f6fa" stroke="#a06400" strokeWidth="1"/>
            <text x="40" y="744" fontFamily="JetBrains Mono" fontSize="11" fill="#a06400" textAnchor="middle" fontWeight="700">7</text>

            <circle cx="40" cy="878" r="9" fill="#f5f6fa" stroke="#0d7a4a" strokeWidth="1"/>
            <text x="40" y="882" fontFamily="JetBrains Mono" fontSize="11" fill="#0d7a4a" textAnchor="middle" fontWeight="700">8</text>
          </svg>
        </div>
      </div>

      <p>
        <i>* Without Kargo, promotion is something your pipelines do — not something your platform can see, govern, and control.</i>
      </p>

      <h1>With Kargo</h1>

      <p>
        <strong>Promotion</strong> is modeled and managed as a first-class, <strong>GitOps-native</strong> capability. Features:

        <ul className={styles.bulletList}>
          <li>A <strong>single control plane</strong> for multi-environment promotion</li>
          <li>Declarative promotion modeled as <strong>desired state</strong></li>
          <li>The exact same <strong>application version and configuration</strong> promoted between environments</li>
          <li><strong>Fully reproducible</strong> rollouts and rollbacks</li>
          <li>Complete visibility into <strong>what changed, when, and why</strong></li>
        </ul>
      </p>

      <img
        src="/img/kargo-ui.png"
        alt="Kargo UI screenshot"
        className={styles.zoomImage}
        onClick={() => setIsOpen(true)}
      />

      {isOpen && (
        <div className={styles.modalOverlay} onClick={() => setIsOpen(false)}>
          <img
            src="/img/kargo-ui.png"
            alt="Kargo UI screenshot"
            className={styles.modalImage}
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}

      <p>
        <i>* Promotion becomes infrastructure — not behavior embedded in CI pipelines.</i>
        <br />
        <i>* Kargo provides structure, visibility, and reproducibility to the <strong>last mile of delivery</strong>.</i>
      </p>

      <h1>Who Kargo Is For</h1>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "1rem", margin: "1.5rem 0" }}>
        <div className={styles.audienceCard} style={{ borderTop: "5px solid #6366f1" }}>
          <p>Platform teams standardizing how versions move between environments</p>
        </div>
        <div className={styles.audienceCard} style={{ borderTop: "5px solid #0ea5e9" }}>
          <p>Organizations operating multi-stage GitOps workflows</p>
        </div>
        <div className={styles.audienceCard} style={{ borderTop: "5px solid #10b981" }}>
          <p>Teams scaling past CI-embedded promotion logic</p>
        </div>
        <div className={styles.audienceCard} style={{ borderTop: "5px solid #f59e0b" }}>
          <p>Argo CD users extending GitOps beyond deployment</p>
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

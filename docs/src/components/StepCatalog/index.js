import React, {useEffect, useState} from 'react';
import Link from '@docusaurus/Link';
import {useLocation} from '@docusaurus/router';
import {
  useCurrentSidebarCategory,
  useDocsVersion,
} from '@docusaurus/plugin-content-docs/client';
import {
  SiArgo,
  SiGit,
  SiGithub,
  SiHelm,
  SiJfrog,
  SiJira,
  SiKubernetes,
  SiOpencontainersinitiative,
  SiOpentofu,
} from 'react-icons/si';
import {
  LuCircleX,
  LuFileJson,
  LuFiles,
  LuGlobe,
  LuLayoutGrid,
  LuList,
  LuMerge,
  LuMessageSquareText,
  LuScrollText,
  LuTag,
  LuTags,
  LuTextCursorInput,
  LuTicket,
  LuUserCheck,
  LuWorkflow,
} from 'react-icons/lu';
import {isBeta, isEnterprise} from '@site/tags';
import styles from './styles.module.css';

// Icon for a step, picked by name prefix. Falls back to a generic icon.
const ICONS = [
  [/^git-/, SiGit],
  [/^(gh|gha|github)-/, SiGithub],
  [/^helm-/, SiHelm],
  [/^kustomize-/, SiKubernetes],
  [/^argocd-/, SiArgo],
  [/^(tf|hcl)-/, SiOpentofu],
  [/^jira$/, SiJira],
  [/^jfrog-/, SiJfrog],
  [/^oci-/, SiOpencontainersinitiative],
  [/^http/, LuGlobe],
  [/^(json|yaml|toml)-/, LuFileJson],
  [/^(copy|delete|file-write|tar|untar)$/, LuFiles],
  [/^wait-for-approval$/, LuUserCheck],
  [/^get-user-input$/, LuTextCursorInput],
  [/^send-message$/, LuMessageSquareText],
  [/^record-audit-event$/, LuScrollText],
  [/^set-metadata$/, LuTags],
  [/^set-freight-alias$/, LuTag],
  [/^compose-output$/, LuMerge],
  [/^fail$/, LuCircleX],
  [/^snow-/, LuTicket],
];

function StepIcon({name}) {
  const Icon = ICONS.find(([pattern]) => pattern.test(name))?.[1] ?? LuWorkflow;
  return <Icon className={styles.icon} aria-hidden="true" />;
}

// Descriptions come from front matter and may contain `code` spans.
function Description({text, className}) {
  return (
    <p className={className}>
      {text
        .split(/`([^`]+)`/)
        .map((part, i) => (i % 2 ? <code key={i}>{part}</code> : part))}
    </p>
  );
}

function StepCard({step}) {
  return (
    <Link to={step.href} className={styles.card}>
      <div className={styles.cardHeader}>
        <StepIcon name={step.label} />
        <span className={styles.cardTitle}>{step.label}</span>
      </div>
      <Description text={step.description} className={styles.cardDescription} />
      {/* Always rendered (even when empty) so every card is the same height. */}
      <Badges step={step} className={styles.badges} />
    </Link>
  );
}

function Badges({step, className}) {
  return (
    <div className={className}>
      {step.enterprise && <span className={styles.entBadge}>Enterprise</span>}
      {step.beta && <span className={styles.betaBadge}>Beta</span>}
    </div>
  );
}

// List view: one row per step, with the description clamped to two lines.
function StepRow({step}) {
  return (
    <Link to={step.href} className={styles.row}>
      <span className={styles.rowName}>
        <StepIcon name={step.label} />
        {step.label}
      </span>
      <Description text={step.description} className={styles.rowDescription} />
      <Badges step={step} className={styles.rowBadges} />
    </Link>
  );
}

const VIEWS = [
  {key: 'grid', label: 'Grid view', Icon: LuLayoutGrid},
  {key: 'list', label: 'List view', Icon: LuList},
];
const VIEW_STORAGE_KEY = 'kargo.promotionStepCatalog.view';

export default function StepCatalog() {
  const category = useCurrentSidebarCategory();
  const groups = category.items.filter((item) => item.type === 'category');
  const [query, setQuery] = useState('');
  const [activeGroup, setActiveGroup] = useState(null);
  const [view, setView] = useState('grid');

  // Filters and view can be preset from the URL (e.g. ?q=helm&group=Git or
  // ?view=list); otherwise the view is the one last chosen in this browser.
  // Applied after the first render so it matches the pre-built HTML, which
  // has no filters and uses the grid view.
  const {search} = useLocation();
  useEffect(() => {
    const params = new URLSearchParams(search);
    setQuery(params.get('q') ?? '');
    setActiveGroup(params.get('group'));
    const preferred =
      params.get('view') ?? window.localStorage.getItem(VIEW_STORAGE_KEY);
    setView(preferred === 'list' ? 'list' : 'grid');
  }, [search]);

  const chooseView = (key) => {
    setView(key);
    window.localStorage.setItem(VIEW_STORAGE_KEY, key);
  };
  const {docs} = useDocsVersion();
  const steps = groups.flatMap((group) =>
    group.items.map((item) => ({
      label: item.label,
      href: item.href,
      description: docs[item.docId]?.description ?? '',
      group: group.label,
      enterprise: isEnterprise(item.docId),
      beta: isBeta(item.docId),
    })),
  );

  const q = query.trim().toLowerCase();
  const visible = steps.filter(
    (step) =>
      (!activeGroup || step.group === activeGroup) &&
      (!q ||
        step.label.toLowerCase().includes(q) ||
        step.description.toLowerCase().includes(q)),
  );
  return (
    <div>
      <div className={styles.toolbar}>
        <input
          type="search"
          aria-label="Search promotion steps"
          className={styles.search}
          placeholder="Search steps by name or description…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <select
          aria-label="Filter by group"
          className={styles.select}
          value={activeGroup ?? ''}
          onChange={(e) => setActiveGroup(e.target.value || null)}
        >
          <option value="">All groups</option>
          {groups.map((group) => (
            <option key={group.label} value={group.label}>
              {group.label}
            </option>
          ))}
        </select>
        <div className={styles.viewToggle} role="group" aria-label="Layout">
          {VIEWS.map(({key, label, Icon}) => (
            <button
              key={key}
              type="button"
              aria-label={label}
              aria-pressed={view === key}
              title={label}
              className={
                view === key ? styles.viewButtonActive : styles.viewButton
              }
              onClick={() => chooseView(key)}
            >
              <Icon aria-hidden="true" />
            </button>
          ))}
        </div>
      </div>
      <p className={styles.count} aria-live="polite">
        Showing {visible.length} of {steps.length} steps
      </p>
      {groups.map((group) => {
        const groupSteps = visible.filter((step) => step.group === group.label);
        if (!groupSteps.length) {
          return null;
        }
        return (
          <section key={group.label}>
            <h2 className={styles.groupHeading}>
              {group.label}
              <span className={styles.groupCount}>{groupSteps.length}</span>
            </h2>
            {view === 'list' ? (
              <div className={styles.list}>
                {groupSteps.map((step) => (
                  <StepRow key={step.href} step={step} />
                ))}
              </div>
            ) : (
              <div className={styles.grid}>
                {groupSteps.map((step) => (
                  <StepCard key={step.href} step={step} />
                ))}
              </div>
            )}
          </section>
        );
      })}
      {!visible.length && (
        <p className={styles.empty}>No steps match your search.</p>
      )}
    </div>
  );
}

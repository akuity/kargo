import {
  faCircleNotch,
  faDiagramProject,
  faStar,
  faTriangleExclamation
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';
import classNames from 'classnames';
import { formatDistanceToNow } from 'date-fns';
import { CSSProperties } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import type { Project } from '@ui/gen/api/v2/models';

import {
  HealthSummary,
  ProjectReadiness,
  healthSummary,
  projectAccent,
  projectReadiness
} from './project-card-utils';

// The card's border says how setting the project up went, as the Warehouse
// node's does. A broken project keeps its border under the pointer, so
// hovering it does not hide that it is broken.
const cardBorder: Record<ProjectReadiness['state'], string> = {
  ready:
    'border-[color:var(--kargo-color-border-secondary)] hover:border-[color:var(--kargo-tile-hover-border,var(--kargo-color-border))]',
  syncing: 'border-[color:color-mix(in_srgb,var(--kargo-color-info)_50%,transparent)]',
  failed: 'border-[color:color-mix(in_srgb,var(--kargo-color-error)_50%,transparent)]'
};

// The same surface as the pipeline's nodes -- the container color, a quiet
// border and the nodes' own shadow, which only light sets -- so the list and
// the pipeline it opens onto read as one product. Shared with the card's
// placeholder, so the two take the same room.
const cardSurface = classNames(
  'relative flex min-w-0 flex-col rounded-[10px] border border-solid px-4 pb-3 pt-3.5',
  'bg-[color:var(--kargo-color-bg-container)]',
  'shadow-[shadow:var(--stage-node-shadow,0_0_#0000)]'
);

// The stats sit in a well cut into the card, down to the color of the page
// behind it -- the same well a Stage carries its freight in. On dark the page
// sits far below the card, so the well is lifted a little off it.
const statsWell = classNames(
  'mt-3.5 grid auto-cols-[minmax(0,1fr)] grid-flow-col gap-4 rounded-lg px-3 py-2.5',
  'bg-[color:var(--kargo-color-bg-layout)]',
  'dark:bg-[color:color-mix(in_srgb,var(--kargo-color-bg-layout)_70%,var(--kargo-color-bg-container))]'
);

const readinessColor: Record<ProjectReadiness['state'], string> = {
  ready: '',
  syncing: 'text-[color:var(--kargo-color-info-text)]',
  failed: 'text-[color:var(--kargo-color-error-text)]'
};

// The star's color goes on the icon rather than the button: AntD recolors a
// text button under the pointer, and a gold star turning grey as it is
// reached for reads as having been unstarred. Not starred, it stays in the
// background, and comes forward only when the keyboard reaches it. AntD
// resets a button's icon to `color: inherit` with a selector that outranks
// these, hence the `!`.
const starColor = (starred: boolean) =>
  starred
    ? '[&_svg]:!text-[#f2a600] dark:[&_svg]:!text-[#ffc53d]'
    : '[&_svg]:!text-[color:var(--kargo-color-text-quaternary)] [&:focus-visible_svg]:!text-[color:var(--kargo-color-text)]';

type ProjectItemProps = {
  project?: Project;
  starred: boolean;
  onToggleStar: (uid: string) => void;
};

// ProjectItem is a project's card on the project list: who it is, how its
// Stages, Warehouses and Targets are faring, and whether the controller has
// finished setting it up. The whole card opens the project; the star is the
// one thing on it that does anything else.
export const ProjectItem = ({ project, starred, onToggleStar }: ProjectItemProps) => {
  const name = project?.metadata?.name || '';
  const description = project?.metadata?.annotations?.[DESCRIPTION_ANNOTATION_KEY];
  const stats = project?.status?.stats;
  const readiness = projectReadiness(project);
  const createdAt = project?.metadata?.creationTimestamp;
  const created = createdAt ? new Date(createdAt) : undefined;

  const meters: { label: string; summary: HealthSummary }[] = [
    { label: 'Stages', summary: healthSummary(stats?.stages) },
    { label: 'Warehouses', summary: healthSummary(stats?.warehouses) }
  ];
  // The controller leaves Targets out for a project that promotes to none, so
  // only a project with a fleet has a third meter.
  if (stats?.targets?.count) {
    meters.push({ label: 'Targets', summary: healthSummary(stats.targets) });
  }

  return (
    <div
      className={classNames(
        'transition-[border-color,box-shadow]',
        cardSurface,
        'hover:shadow-[0_1px_2px_0_rgba(0,0,0,0.03),0_6px_16px_-4px_rgba(0,0,0,0.08)]',
        'dark:hover:shadow-none',
        cardBorder[readiness.state]
      )}
      style={{ '--project-accent': projectAccent(name) } as CSSProperties}
    >
      {/* The tile, the name and the star share the first row and center on it,
          so the name sits level with the tile whether or not a description
          follows. The description goes under the name, pulled up into the
          room the row leaves below it. */}
      <div className='grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-3'>
        {/* The project's color, stated once and only here, the way a Warehouse
            states its own in the tile beside its name. */}
        <span
          aria-hidden
          className={classNames(
            'flex size-[34px] flex-none items-center justify-center rounded-lg text-[15px]',
            'bg-[color:color-mix(in_srgb,var(--project-accent)_18%,transparent)]',
            'text-[color:color-mix(in_srgb,var(--project-accent)_55%,black)]',
            'dark:text-[color:color-mix(in_srgb,var(--project-accent)_55%,white)]'
          )}
        >
          <FontAwesomeIcon icon={faDiagramProject} />
        </span>
        {/* The name is the card's link, stretched over the whole card by its
            ::after, so anywhere on it opens the project. It still reads as a
            title: no color of its own, and nothing changes under the pointer. */}
        <Link
          to={generatePath(paths.project, { name })}
          className={classNames(
            'block truncate text-[15px] font-semibold leading-5 no-underline',
            'text-[color:var(--kargo-color-text-heading)]',
            'hover:text-[color:var(--kargo-color-text-heading)] hover:no-underline',
            'active:text-[color:var(--kargo-color-text-heading)]',
            "after:absolute after:inset-0 after:rounded-[10px] after:content-['']",
            'focus-visible:outline-none focus-visible:after:outline focus-visible:after:outline-2',
            'focus-visible:after:outline-offset-2 focus-visible:after:outline-[color:var(--kargo-color-link)]'
          )}
          title={name}
        >
          {name}
        </Link>
        {/* Above the stretched link, so it answers its own click. AntD sizes
            the button with classes that tie with these, hence the `!`. */}
        <Button
          type='text'
          size='small'
          className={classNames(
            'relative z-[1] -mr-1.5 !h-6 !w-6 !min-w-6 !p-0 !text-[12px]',
            '[&_svg]:transition-colors',
            starColor(starred)
          )}
          aria-label={`${starred ? 'Unstar' : 'Star'} ${name}`}
          aria-pressed={starred}
          icon={<FontAwesomeIcon icon={faStar} />}
          onClick={() => onToggleStar(project?.metadata?.uid || '')}
        />
        {description && (
          <p
            className={classNames(
              'col-[2/-1] -mt-[5px] mb-0 line-clamp-2 text-xs leading-[18px] [overflow-wrap:anywhere]',
              'text-[color:var(--kargo-color-text-secondary)]'
            )}
          >
            {description}
          </p>
        )}
      </div>

      <div className={statsWell}>
        {meters.map(({ label, summary }) => (
          <HealthMeter key={label} label={label} summary={summary} />
        ))}
      </div>

      {/* Pinned to the bottom of the card, so footers line up along a row of
          cards whose descriptions run to different lengths. */}
      <div
        className={classNames(
          'mt-auto flex min-h-4 items-center justify-between gap-2 pt-2.5 text-[11px] leading-4',
          'text-[color:var(--kargo-color-text-tertiary)]'
        )}
      >
        {created && <span>Created {formatDistanceToNow(created, { addSuffix: true })}</span>}
        {readiness.state !== 'ready' && (
          <Tooltip title={readiness.message}>
            {/* Above the stretched link, so its tooltip can say why. */}
            <span
              className={classNames(
                'relative z-[1] ml-auto inline-flex flex-none items-center gap-[5px] font-semibold',
                readinessColor[readiness.state]
              )}
            >
              {readiness.state === 'syncing' ? (
                <>
                  <FontAwesomeIcon icon={faCircleNotch} spin />
                  Setting up
                </>
              ) : (
                <>
                  <FontAwesomeIcon icon={faTriangleExclamation} />
                  Not ready
                </>
              )}
            </span>
          </Tooltip>
        )}
      </div>
    </div>
  );
};

const placeholderFill = 'bg-[color:var(--kargo-color-fill-secondary)]';
const placeholderBar = classNames('rounded', placeholderFill);

// ProjectItemSkeleton is a card's outline at the card's own size -- the tile
// and the name, a well with two meters, then the footer -- for the list to
// show while its projects load. It leaves out the description, which not
// every project has. Dimmed as the list's previous results are while the next
// ones load; on the card rather than the grid, whose opacity the pulse drives.
export const ProjectItemSkeleton = () => (
  <div
    aria-hidden
    className={classNames(
      cardSurface,
      'border-[color:var(--kargo-color-border-secondary)] opacity-60'
    )}
  >
    <div className='flex items-center gap-x-3'>
      <div className={classNames(placeholderFill, 'size-[34px] flex-none rounded-lg')} />
      <div className={classNames(placeholderBar, 'h-3 w-2/5')} />
    </div>
    <div className={statsWell}>
      {['w-3/5', 'w-4/5'].map((width) => (
        <div key={width} className='min-w-0'>
          <div className='mb-0.5 flex h-3 items-center'>
            <div className={classNames(placeholderBar, 'h-2', width)} />
          </div>
          <div className='flex h-5 items-center'>
            <div className={classNames(placeholderBar, 'h-3 w-8')} />
          </div>
        </div>
      ))}
    </div>
    <div className='mt-auto flex min-h-4 items-center pb-1 pt-3.5'>
      <div className={classNames(placeholderBar, 'h-2 w-1/3')} />
    </div>
  </div>
);

// HealthMeter states one kind of resource: how many of them are healthy, out
// of how many.
const HealthMeter = ({ label, summary }: { label: string; summary: HealthSummary }) => (
  <div className='min-w-0'>
    {/* The same caption a Stage sets over its Current Freight. */}
    <div className='mb-0.5 truncate text-[10px] uppercase leading-3 tracking-[0.08em] text-[color:var(--kargo-color-text-tertiary)]'>
      {label}
    </div>
    {summary.count ? (
      <div
        className={classNames('whitespace-nowrap text-sm font-semibold tabular-nums', {
          'text-[color:var(--kargo-color-success-text)]': summary.allHealthy
        })}
      >
        <span aria-hidden>
          {summary.healthy}
          <span className='font-normal text-[color:var(--kargo-color-text-tertiary)]'>
            /{summary.count}
          </span>
        </span>
        <span className='sr-only'>
          {summary.healthy} of {summary.count} healthy
        </span>
      </div>
    ) : (
      <div className='whitespace-nowrap text-xs leading-5 text-[color:var(--kargo-color-text-tertiary)]'>
        None
      </div>
    )}
  </div>
);

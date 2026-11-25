import classNames from 'classnames';
import { CSSProperties, useLayoutEffect, useRef, useState } from 'react';

import { floatingSurfaceBorderColor, floatingSurfaceClassName } from './floating-surface';
import { FreightTimelineSkeleton } from './freight/freight-timeline-skeleton';
import { RANK_SEPARATION } from './graph/layout-graph';
import { repoSubscriptionSizer, stageSizer, warehouseSizer } from './graph/node-sizer';

const barClassName = 'rounded bg-[color:var(--kargo-color-fill-secondary)]';

const outlineClassName =
  'rounded-lg border border-solid bg-[color:var(--kargo-color-bg-container)] border-[color:var(--kargo-color-border-secondary)]';

const stageLines = ['w-4/5', 'w-3/5', 'w-2/3', 'w-1/2', 'w-3/4', 'w-2/5'];

// A subscription feeding a Warehouse feeding two Stages, at the graph's own
// node sizes, each with a title and a few lines of content.
const nodes = [
  { size: repoSubscriptionSizer.size(), lines: ['w-4/5'] },
  { size: warehouseSizer.size(), lines: ['w-3/4', 'w-1/2'] },
  { size: stageSizer.size(), lines: stageLines },
  { size: stageSizer.size(), lines: stageLines }
];

const chainWidth =
  nodes.reduce((width, node) => width + node.size.width, 0) + (nodes.length - 1) * RANK_SEPARATION;
const chainHeight = Math.max(...nodes.map((node) => node.size.height));

// Room kept around the chain when it is scaled down to fit.
const FIT_MARGIN = 48;
// The chain is never scaled past full size, nor below the graph's own minimum
// zoom.
const MIN_SCALE = 0.2;

// useFitScale scales the chain down, like the graph's fit to view, so it
// never runs off a small screen. The scale is undefined until the space is
// first measured.
const useFitScale = () => {
  const ref = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState<number>();

  useLayoutEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }

    const observer = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect;
      setScale(
        Math.max(
          MIN_SCALE,
          Math.min(
            1,
            (width - 2 * FIT_MARGIN) / chainWidth,
            (height - 2 * FIT_MARGIN) / chainHeight
          )
        )
      );
    });

    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return { ref, scale };
};

const NodePlaceholder = (props: { size: { width: number; height: number }; lines: string[] }) => (
  <div
    className={classNames(outlineClassName, 'box-border shrink-0 flex flex-col gap-2 p-3')}
    style={{ width: props.size.width, height: props.size.height }}
  >
    <div className={classNames(barClassName, 'h-3 w-1/2')} />
    {props.lines.map((width, i) => (
      <div key={i} className={classNames(barClassName, 'h-2.5', width)} />
    ))}
  </div>
);

const Edge = () => (
  <div
    className='shrink-0 h-px bg-[color:var(--kargo-color-border)]'
    style={{ width: RANK_SEPARATION }}
  />
);

// PipelineSkeleton stands in for the pipeline while Stages load: nodes the
// graph's size, as far apart as the graph lays them out, centered in the space
// it is given and scaled down when that space is too small. Centered below the
// freight timeline, it sits where the graph fits a pipeline of its shape. It
// lets clicks through, since it may lie over the toolbar.
export const PipelineSkeleton = (props: {
  // draw the graph's dotted canvas behind it, as the graph view will
  canvas?: boolean;
  className?: string;
  style?: CSSProperties;
}) => {
  const { ref, scale } = useFitScale();

  return (
    <div
      ref={ref}
      role='status'
      aria-label='Loading pipeline'
      className={classNames(
        'flex items-center justify-center overflow-hidden pointer-events-none',
        props.canvas && 'kargo-pipeline-canvas',
        props.className
      )}
      style={props.style}
    >
      <div
        aria-hidden
        className={classNames(
          'shrink-0 flex items-center motion-safe:animate-pulse',
          // The first measurement arrives after the first paint; until then
          // the chain would show at full size, running off a small screen.
          scale === undefined && 'invisible'
        )}
        style={{ transform: `scale(${scale ?? 1})` }}
      >
        {nodes.map((node, i) => (
          <div key={i} className='flex items-center'>
            {i > 0 && <Edge />}
            <NodePlaceholder size={node.size} lines={node.lines} />
          </div>
        ))}
      </div>
    </div>
  );
};

// PipelinePageSkeleton stands in for the whole page while the project and its
// freight load: the freight timeline, the toolbar's outline under it, and the
// pipeline centered below the timeline, each where it will appear. In the
// graph view the dotted canvas runs under the freight timeline, as the graph
// does.
export const PipelinePageSkeleton = (props: { canvas: boolean }) => (
  <div
    className={classNames(
      'overflow-hidden h-full flex flex-col',
      props.canvas && 'kargo-pipeline-canvas'
    )}
  >
    <FreightTimelineSkeleton />
    <div className='relative flex-1'>
      <PipelineSkeleton className='absolute inset-0' />
      <div
        aria-hidden
        className={classNames(
          floatingSurfaceClassName,
          'absolute top-2 left-2 box-border h-12 w-[620px] max-w-[calc(100%-16px)] rounded-lg border border-solid motion-safe:animate-pulse'
        )}
        style={{ borderColor: floatingSurfaceBorderColor }}
      />
    </div>
  </div>
);

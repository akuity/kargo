import classNames from 'classnames';

import {
  floatingSurfaceBorderColor,
  floatingSurfaceClassName
} from '@ui/features/project/pipelines/floating-surface';

import { ARROW_BUTTON_WIDTH, CARD_GAP, MIN_CARD_WIDTH } from './use-freight-carousel';

// More than the widest screen fits; the rest are clipped.
const PLACEHOLDER_COUNT = 24;

// Cards that open a Warehouse run and so carry its title: a run of five, then
// another.
const TITLED = new Set([0, 5]);

const barClassName = 'rounded bg-[color:var(--kargo-color-fill-secondary)]';

// A freight card's outline at the card's own size -- the alias, two artifact
// rows, then the age -- so the strip keeps its height once freight arrives.
const FreightCardPlaceholder = () => (
  <div className='flex flex-col rounded-md border border-solid py-1.5 px-2 bg-white border-gray-200 shadow-sm dark:shadow-none dark:bg-[#262626] dark:border-[#3d3d3d]'>
    <div className='flex items-center h-5'>
      <div className={classNames(barClassName, 'h-2.5 w-3/5')} />
    </div>
    <div className='flex flex-col gap-[2px] mt-1 min-h-[42px]'>
      {['w-4/5', 'w-3/5'].map((width) => (
        <div key={width} className='flex items-center gap-1.5 h-5'>
          <div className={classNames(barClassName, 'w-3.5 h-3.5 shrink-0')} />
          <div className={classNames(barClassName, 'h-2.5', width)} />
        </div>
      ))}
    </div>
    <div className='pt-1'>
      <div className='flex items-center h-[18px]'>
        <div className={classNames(barClassName, 'h-2 w-1/4')} />
      </div>
    </div>
  </div>
);

// A card under the row Warehouse titles sit in, titled when it opens a run.
const FreightPlaceholder = (props: { titled: boolean }) => (
  <div className='flex flex-col gap-1 min-w-0'>
    <div className='flex items-center gap-1.5 h-5'>
      {props.titled && (
        <>
          <div className={classNames(barClassName, 'w-4 h-4 shrink-0')} />
          <div className={classNames(barClassName, 'h-2.5 w-1/2')} />
        </>
      )}
    </div>
    <FreightCardPlaceholder />
  </div>
);

// FreightTimelineSkeleton stands in for the freight timeline while freight
// loads, on the same surface. Its grid follows the carousel's sizing -- as
// many cards as fit at the minimum width, stretched to fill the row -- and
// puts the rest on rows of no height, out of sight.
export const FreightTimelineSkeleton = () => (
  <div
    role='status'
    aria-label='Loading freight'
    className={classNames(floatingSurfaceClassName, 'py-2 flex relative z-20')}
    style={{ borderBottom: `1px solid ${floatingSurfaceBorderColor}` }}
  >
    <div className='shrink-0 mx-1' style={{ width: ARROW_BUTTON_WIDTH }} />
    <div
      aria-hidden
      className='flex-1 min-w-0 grid overflow-hidden motion-safe:animate-pulse'
      style={{
        gridTemplateColumns: `repeat(auto-fill, minmax(${MIN_CARD_WIDTH}px, 1fr))`,
        gridTemplateRows: 'auto',
        gridAutoRows: 0,
        columnGap: CARD_GAP
      }}
    >
      {Array.from({ length: PLACEHOLDER_COUNT }, (_, i) => (
        <FreightPlaceholder key={i} titled={TITLED.has(i)} />
      ))}
    </div>
    <div className='shrink-0 mx-1' style={{ width: ARROW_BUTTON_WIDTH }} />
  </div>
);

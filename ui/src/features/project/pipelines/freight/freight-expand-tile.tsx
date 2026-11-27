import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

type Props = {
  count: number;
};

// FreightExpandTile stands in for a run of freight that "Hide unused freights"
// folds away: a narrow dashed tile labelled "+N" that turns hiding off.
export const FreightExpandTile = ({ count }: Props) => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  // Showing them turns hiding off for the whole timeline, not only this run,
  // and the label says so.
  const label = `Show all unused freight (${count} hidden here)`;

  return (
    <button
      type='button'
      aria-label={label}
      title={label}
      onClick={() =>
        freightTimelineControllerContext?.setPreferredFilter({
          ...freightTimelineControllerContext?.preferredFilter,
          hideUnusedFreights: false
        })
      }
      // borders as on a freight card, and the same step up on hover, so the
      // tile reads as one of the cards
      className='shrink-0 w-7 p-0 flex items-center justify-center rounded-md border border-dashed cursor-pointer bg-transparent font-[inherit] text-[12px] border-gray-200 text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:border-[#3d3d3d] dark:text-neutral-400 dark:hover:border-[#4d4d4d] dark:hover:text-neutral-200'
    >
      <span className='[writing-mode:vertical-rl] rotate-180'>+{count}</span>
    </button>
  );
};

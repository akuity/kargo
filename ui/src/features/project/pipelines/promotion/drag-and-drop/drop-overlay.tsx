import { useDndContext } from '@dnd-kit/core';
import classNames from 'classnames';

import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v2/models';

import { isPromotionWindowClosed, promotionWindowReopensLabel } from '../promotion-window';

type Props = {
  isOver: boolean;
  // whether the Stage requests Freight from the dragged Freight's Warehouse
  eligible: boolean;
  stage: Stage;
};

// The fill is translucent and blurs the freight underneath, so it shows through
// as a shape without competing with the slot's own text. Each state only picks
// the color mixed into the card's, and how much of it. No border: the graph
// zooms, and a thin line breaks up when zoomed out.
const fillClassName =
  'bg-[color:color-mix(in_srgb,color-mix(in_srgb,var(--drop-color)_var(--drop-tint),var(--kargo-color-bg-container))_75%,transparent)]';

// One set per state, never stacked: two utilities setting the same property
// on one element are settled by stylesheet order, not class order.
const stateClassNames = {
  // neutral: a light grey on light, darkened toward black on dark
  promote: '[--drop-color:#f0f0f0] [--drop-tint:60%] dark:[--drop-color:#000]',
  // only a hint of red: the red title carries the message
  reject: '[--drop-color:var(--kargo-color-error)] [--drop-tint:10%]',
  // a schedule, not an error -- neutral rather than red
  blocked: '[--drop-color:var(--kargo-color-bg-layout)] [--drop-tint:70%]'
};

// Covers the Stage's freight well -- not the whole card -- while Freight is
// being dragged, so the header and status stay readable.
export const DropOverlay = ({ isOver, eligible, stage }: Props) => {
  const dndContext = useDndContext();

  const active = dndContext.active;
  const isDragging = Boolean(active);

  const windowClosed = isPromotionWindowClosed(stage);

  // Every eligible stage shows its slot while dragging so the user sees where
  // the freight can go. An ineligible one only says why when hovered --
  // otherwise every unconnected stage would flash red at once. The node fades
  // those out instead.
  //
  // A closed promotion window is different: the stage is connected, so the user
  // would otherwise aim for it. Mark it as blocked for the whole drag.
  const showBlocked = isDragging && eligible && windowClosed;
  const showPromote = isDragging && eligible && !windowClosed;
  const showReject = isDragging && !eligible && isOver;

  const controlFlow = isStageControlFlow(stage);

  let title = controlFlow ? 'Drop to promote downstream' : 'Drop to promote';
  let detail: string | undefined;

  if (showBlocked) {
    title = 'Promotion window closed';
    detail = promotionWindowReopensLabel(stage);
  } else if (showReject) {
    title = "Can't promote here";
    detail = `Not connected to the ${active?.data.current?.originName} Warehouse`;
  }

  const visible = showPromote || showReject || showBlocked;

  return (
    <div
      className={classNames(
        'pointer-events-none absolute inset-0 z-10 flex flex-col items-center justify-center',
        'gap-0.5 rounded-lg px-2.5 text-center',
        'text-[color:var(--kargo-color-text-secondary)]',
        fillClassName,
        // It hides at once, fading only on the way in: its text switches back
        // to the default the moment it hides, and a fade-out would flash that
        // text. Nothing to blur while hidden, either -- that spares every node
        // the filter outside a drag.
        visible
          ? 'backdrop-blur-[3px] transition-[opacity,background-color] duration-100'
          : 'opacity-0',
        showPromote && stateClassNames.promote,
        showReject && stateClassNames.reject,
        showBlocked && stateClassNames.blocked
      )}
    >
      <span
        className={classNames(
          'text-xs font-semibold leading-4',
          showReject && 'text-[color:var(--kargo-color-error)]'
        )}
      >
        {title}
      </span>
      {detail && (
        <span className='max-w-full truncate text-[11px] leading-[14px] text-[color:var(--kargo-color-text-tertiary)]'>
          {detail}
        </span>
      )}
    </div>
  );
};

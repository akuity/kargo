import classNames from 'classnames';
import { ReactNode, useLayoutEffect, useRef, useState } from 'react';

import { WarehouseIcon } from '../warehouse-icon';

import { getHeaderShift } from './freight-warehouse-group-utils';

type FreightWarehouseGroupProps = {
  warehouse: string;
  // the Warehouse's color in the pipeline graph, if it has one
  color?: string;
  // how far the strip is paged, and how wide its cards are, so the header can
  // follow the group's first card in view
  offset: number;
  cardWidth: number;
  // false leaves the header out altogether
  withHeader: boolean;
  // makes the header a button showing only this Warehouse's freight
  onShowOnly?: () => void;
  // the freight timeline shows only this Warehouse's freight
  focused?: boolean;
  children: ReactNode;
};

const titleButtonClassName = classNames(
  // the header's font rather than the browser's button font
  'p-0 border-0 bg-transparent cursor-pointer [font:inherit]',
  'flex items-center gap-1.5 min-w-0',
  'text-[color:inherit] hover:text-[color:var(--kargo-color-text)]'
);

// FreightWarehouseGroup holds a run of consecutive freight from one Warehouse
// under a header naming the Warehouse.
export const FreightWarehouseGroup = (props: FreightWarehouseGroupProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const [bounds, setBounds] = useState({ left: 0, width: 0 });

  // Where the group sits in the strip changes with everything before it, so
  // it is measured after every render.
  useLayoutEffect(() => {
    const section = ref.current;
    if (!section) return;

    const left = section.offsetLeft;
    const width = section.offsetWidth;

    setBounds((prev) => (prev.left === left && prev.width === width ? prev : { left, width }));
  });

  const shift = getHeaderShift(props.offset, bounds.left, bounds.width, props.cardWidth);

  const title = (
    <>
      <WarehouseIcon color={props.color} />
      <span className='truncate'>{props.warehouse}</span>
    </>
  );

  return (
    <div
      ref={ref}
      role='group'
      aria-label={props.warehouse ? `Freight from Warehouse ${props.warehouse}` : undefined}
      className='flex flex-col gap-1 shrink-0'
    >
      {/* Zero width, stretched to the group's: a long name never makes the
          group wider than its cards, which would throw off paging. Shown
          alone, the group can fit the whole name. */}
      {props.withHeader && (
        <div className='w-0 min-w-full'>
          <div
            className={classNames(
              // the width it may take moves with the header, so it never
              // runs into the next group's while sliding back
              'flex items-center gap-1.5 h-5 text-xs font-semibold transition-[transform,max-width] duration-300 ease-out',
              props.focused
                ? 'text-[color:var(--kargo-color-text)]'
                : 'text-[color:var(--kargo-color-text-tertiary)]'
            )}
            style={{
              transform: `translateX(${shift}px)`,
              ...(props.focused
                ? { width: 'max-content' }
                : { maxWidth: `calc(100% - ${shift}px)` })
            }}
            title={props.warehouse}
          >
            {props.onShowOnly ? (
              <button
                type='button'
                aria-label={`Show only freight from Warehouse ${props.warehouse}`}
                className={titleButtonClassName}
                onClick={props.onShowOnly}
              >
                {title}
              </button>
            ) : (
              title
            )}
          </div>
        </div>
      )}
      <div className='flex flex-1 gap-1'>{props.children}</div>
    </div>
  );
};

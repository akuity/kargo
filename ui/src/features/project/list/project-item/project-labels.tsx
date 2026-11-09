import { Flex, Popover, Tag } from 'antd';
import { KeyboardEvent, SyntheticEvent, useLayoutEffect, useRef, useState } from 'react';

import { fitCount, isActivationKey, visibleLabels } from './visible-labels';

const GAP = 4;

// The tile is a link to the Project, so a click on a label must neither follow
// it nor bubble up to it (Popover content is portalled, but React still
// bubbles synthetic events through the portal to the Link). The same applies
// to Enter on a focused label, which would otherwise activate the Link.
const stop = (e: SyntheticEvent) => {
  e.preventDefault();
  e.stopPropagation();
};

// stopKeys keeps Enter/Space on a focused tag from reaching the Link.
const stopKeys = (e: KeyboardEvent) => {
  if (isActivationKey(e.key)) {
    stop(e);
  }
};

const LabelTag = ({
  label: [key, value],
  onClick
}: {
  label: [string, string];
  onClick?: (key: string, value: string) => void;
}) => (
  <Tag
    bordered={false}
    title={`${key}: ${value}`}
    className='!m-0 min-w-0 max-w-full truncate'
    style={{ cursor: onClick ? 'pointer' : undefined }}
    // Clickable labels act as buttons, so they are reachable and usable from
    // the keyboard too.
    role={onClick ? 'button' : undefined}
    tabIndex={onClick ? 0 : undefined}
    onClick={(e) => {
      stop(e);
      onClick?.(key, value);
    }}
    onKeyDown={
      onClick
        ? (e) => {
            if (isActivationKey(e.key)) {
              stop(e);
              onClick(key, value);
            }
          }
        : undefined
    }
  >
    {key}: {value}
  </Tag>
);

// ProjectLabels shows a Project's labels on one line: as many as fit, then a
// "+N" tag whose popover lists all of them. Clicking a label filters the list
// by it.
export const ProjectLabels = ({
  labels,
  onLabelClick
}: {
  labels?: Record<string, string>;
  onLabelClick?: (key: string, value: string) => void;
}) => {
  const pairs = visibleLabels(labels);
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const [count, setCount] = useState(pairs.length);

  // Every tag is rendered once, invisibly, to learn its width; the last
  // measured element is a "+N" placeholder of the widest realistic size.
  useLayoutEffect(() => {
    const container = containerRef.current;
    const measure = measureRef.current;
    if (!container || !measure) {
      return;
    }
    const recompute = () => {
      const widths = Array.from(measure.children).map((el) => el.getBoundingClientRect().width);
      const plusWidth = widths.pop() ?? 0;
      setCount(fitCount(widths, plusWidth, container.clientWidth, GAP));
    };
    recompute();
    const observer = new ResizeObserver(recompute);
    observer.observe(container);
    return () => observer.disconnect();
  }, [labels]);

  if (pairs.length === 0) {
    return null;
  }

  const hidden = pairs.length - count;

  return (
    <div ref={containerRef} className='relative mt-1 overflow-hidden'>
      <div
        ref={measureRef}
        aria-hidden
        className='absolute invisible flex whitespace-nowrap'
        style={{ gap: GAP }}
      >
        {pairs.map((pair) => (
          <LabelTag key={pair[0]} label={pair} />
        ))}
        <Tag bordered={false} className='!m-0'>
          +{pairs.length}
        </Tag>
      </div>
      <Flex gap={GAP} wrap={false}>
        {pairs.slice(0, count).map((pair) => (
          <LabelTag key={pair[0]} label={pair} onClick={onLabelClick} />
        ))}
        {hidden > 0 && (
          <Popover
            // focus lets keyboard users open the list of all labels.
            trigger={['hover', 'focus']}
            content={
              <Flex gap={GAP} wrap style={{ maxWidth: 320 }}>
                {pairs.map((pair) => (
                  <LabelTag key={pair[0]} label={pair} onClick={onLabelClick} />
                ))}
              </Flex>
            }
          >
            <Tag
              bordered={false}
              className='!m-0 shrink-0 cursor-pointer'
              tabIndex={0}
              aria-label={`Show all ${pairs.length} labels`}
              onClick={stop}
              onKeyDown={stopKeys}
            >
              +{hidden}
            </Tag>
          </Popover>
        )}
      </Flex>
    </div>
  );
};

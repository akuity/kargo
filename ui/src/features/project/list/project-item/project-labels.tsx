import { Flex, Popover, Tag } from 'antd';
import { MouseEvent, useLayoutEffect, useRef, useState } from 'react';

import { fitCount, visibleLabels } from './visible-labels';

const GAP = 4;

// The tile is a link to the Project, so a click on a label must neither follow
// it nor bubble up to it (Popover content is portalled, but React still
// bubbles synthetic events through the portal to the Link).
const stop = (e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();
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
    onClick={(e) => {
      stop(e);
      onClick?.(key, value);
    }}
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
            content={
              <Flex gap={GAP} wrap style={{ maxWidth: 320 }}>
                {pairs.map((pair) => (
                  <LabelTag key={pair[0]} label={pair} onClick={onLabelClick} />
                ))}
              </Flex>
            }
          >
            <Tag bordered={false} className='!m-0 shrink-0 cursor-pointer' onClick={stop}>
              +{hidden}
            </Tag>
          </Popover>
        )}
      </Flex>
    </div>
  );
};

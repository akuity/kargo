import { Flex, Tooltip, theme } from 'antd';
import classNames from 'classnames';

import { Mark, MarkTally } from './marks';

// describe reads a tally back as a sentence, so a bar carries a label for
// readers who cannot see its colours.
const describe = (tallies: MarkTally[]): string =>
  tallies.length
    ? tallies.map(({ mark, count }) => `${count} ${mark.short || mark.label}`).join(', ')
    : 'no Targets';

// MarkDot is one category's colour at legend size. A hollow category draws as
// an outline, so "nothing has promoted here yet" reads as an empty slot rather
// than as a status of its own.
export const MarkDot = ({ mark, size = 8 }: { mark: Mark; size?: number }) => (
  <span
    className='inline-block shrink-0 rounded-sm'
    style={{
      width: size,
      height: size,
      background: mark.hollow ? 'transparent' : mark.color,
      border: mark.hollow ? `1px solid ${mark.color}` : undefined
    }}
  />
);

// MarkBar draws a tally as one bar: the share of the fleet in each category,
// in the order the legend lists them.
export const MarkBar = ({
  tallies,
  height = 6,
  className
}: {
  tallies: MarkTally[];
  height?: number;
  className?: string;
}) => {
  const { token } = theme.useToken();
  const total = tallies.reduce((sum, { count }) => sum + count, 0);
  const description = describe(tallies);

  return (
    <Tooltip title={description}>
      <div
        role='img'
        aria-label={description}
        className={classNames('flex overflow-hidden', className)}
        style={{ height, borderRadius: height, background: token.colorFillTertiary }}
      >
        {tallies.map(({ mark, count }) => (
          <div
            key={mark.key}
            style={{
              width: `${(count / total) * 100}%`,
              background: mark.color,
              opacity: mark.hollow ? 0.35 : undefined,
              transition: 'width 0.3s ease'
            }}
          />
        ))}
      </div>
    </Tooltip>
  );
};

// MarkLegend says what the squares mean, a dot and a count per category.
export const MarkLegend = ({ tallies }: { tallies: MarkTally[] }) => (
  <Flex gap={16} wrap align='center'>
    {tallies.map(({ mark, count }) => (
      <Flex key={mark.key} gap={6} align='center'>
        <MarkDot mark={mark} />
        <span className='text-xs whitespace-nowrap'>
          {mark.label} <span className='font-semibold'>{count}</span>
        </span>
      </Flex>
    ))}
  </Flex>
);

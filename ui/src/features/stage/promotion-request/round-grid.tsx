import { theme } from 'antd';

import { getPromotionPhasePresentation } from '@ui/features/common/promotion-status/promotion-phase';

import { ChildPromotion } from './round-detail';

// A round of any size has to fit in a card, so each Target gets a small square
// and the grid wraps. Past this many the squares stop being legible as
// individuals anyway, and the bar above the grid still shows the whole round.
const maxSquares = 600;

/**
 * RoundGrid draws one square per Target of a round, colored by what became of
 * its Promotion. Where the bar above it gives the proportions, the grid gives
 * the texture: six red squares scattered through the round reads differently
 * from six in a row, and that difference is often the whole diagnosis.
 *
 * Squares carry a plain title attribute rather than a tooltip component: a
 * round of hundreds would otherwise mount hundreds of popups to show one.
 */
export const RoundGrid = ({ rows }: { rows: ChildPromotion[] }) => {
  const { token } = theme.useToken();

  if (!rows.length) {
    return null;
  }

  const shown = rows.slice(0, maxSquares);
  const hidden = rows.length - shown.length;

  return (
    <div className='flex flex-wrap gap-1' role='img' aria-label={`${rows.length} Promotions`}>
      {shown.map((row) => (
        <div
          key={row.target}
          title={`${row.target} — ${row.phase}`}
          style={{
            width: 10,
            height: 10,
            borderRadius: 2,
            background: token[getPromotionPhasePresentation(row.phase).colorToken]
          }}
        />
      ))}
      {hidden > 0 && (
        <span className='text-xs ml-1' style={{ color: token.colorTextSecondary }}>
          +{hidden} more
        </span>
      )}
    </div>
  );
};

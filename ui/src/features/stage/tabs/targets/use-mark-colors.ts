import { theme } from 'antd';
import { useMemo } from 'react';

import {
  PromotionPhase,
  promotionPhasePresentations,
  promotionPhases
} from '@ui/features/common/promotion-status/promotion-phase';
import { useStateColors } from '@ui/features/project/targets/shared/use-state-colors';
import { ColorMapHex } from '@ui/features/stage/utils';

import { MarkColors } from './marks';

// The hues a fleet's Freight is painted with, drawn from the palette the
// pipeline paints Stages with so a square on this tab is in the same family as
// the rest of the UI. Fleets run a handful of distinct Freight collections at
// once, so the list need not be long -- it cycles if one ever does.
const freightPalette = [
  ColorMapHex.blue,
  ColorMapHex.orange,
  ColorMapHex.teal,
  ColorMapHex.violet,
  ColorMapHex.amber,
  ColorMapHex.cyan,
  ColorMapHex.pink,
  ColorMapHex.lime
];

// useMarkColors resolves every colour the tab paints with against the live
// theme, once, so the tiles and the grid below them follow dark mode together
// and the pure marking builders never have to know about either.
export const useMarkColors = (): MarkColors => {
  const { token } = theme.useToken();
  const state = useStateColors();

  return useMemo(
    () => ({
      phase: Object.fromEntries(
        promotionPhases.map((phase) => [
          phase,
          token[promotionPhasePresentations[phase].colorToken]
        ])
      ) as Record<PromotionPhase, string>,
      state,
      freight: freightPalette
    }),
    [token, state]
  );
};

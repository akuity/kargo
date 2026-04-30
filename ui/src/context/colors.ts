import { createContext } from 'react';

import { ColorMap } from '@ui/features/stage/utils';

export const ColorContext = createContext(
  {} as {
    stageColorMap: ColorMap;
    warehouseColorMap: ColorMap;
  }
);

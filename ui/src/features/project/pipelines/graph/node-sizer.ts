export const warehouseSizer = {
  size: () => ({ width: 270, height: 100 })
};

export const repoSubscriptionSizer = {
  size: () => ({ width: 250, height: 100 })
};

export const stageSizer = {
  size: () => ({ width: 270, height: 170 })
};

export const stackSizer = {
  size: () => ({ width: 250, height: 100 })
};

export const STACKED_NODE_DUMMY_KEY = '__stacked__';

export const pickMaxSize = (...sizes: { width: number; height: number }[]) => {
  return {
    width: Math.max(...sizes.map((s) => s.width).filter(Boolean)),
    height: Math.max(...sizes.map((s) => s.height).filter(Boolean))
  };
};

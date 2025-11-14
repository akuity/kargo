export const warehouseSizer = {
  size: () => ({ width: 300, height: 100 })
};

export const repoSubscriptionSizer = {
  size: () => ({ width: 300, height: 100 })
};

export const stageSizer = {
  size: () => ({ width: 300, height: 170 })
};

export const stackSizer = {
  size: () => ({ width: 300, height: 100 })
};

export const pickMaxSize = (...sizes: { width: number; height: number }[]) => {
  return {
    width: Math.max(...sizes.map((s) => s.width).filter(Boolean)),
    height: Math.max(...sizes.map((s) => s.height).filter(Boolean))
  };
};

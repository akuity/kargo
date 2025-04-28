export const edgeIndexer = {
  index: (sourceWarehouse: string, sourceStage: string, destStage: string) =>
    `${sourceWarehouse}/${sourceStage}/${destStage}`
};

import { WarehouseExpanded } from '@ui/extend/types';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { ProcessedTagMap } from './types';

export const getStages = (selectedImageData: ProcessedTagMap | undefined): string[] => {
  if (!selectedImageData) return [];

  const stageSet = new Set<string>();

  for (const imageStageMap of Object.values(selectedImageData.tags)) {
    for (const stageName of Object.keys(imageStageMap.stages || {})) {
      stageSet.add(stageName);
    }
  }

  return Array.from(stageSet).sort();
};

const findWarehousesForImageRepo = (repoURL: string, warehouses: WarehouseExpanded[]): string[] => {
  return warehouses
    .filter((w) => w.spec?.subscriptions?.some((s) => s.image?.repoURL === repoURL))
    .map((w) => w.metadata?.name || '')
    .filter(Boolean);
};

export const findStagesForWarehouse = (warehouseName: string, stages: Stage[]): Set<string> => {
  const reachableStages = new Set<string>();
  for (const stage of stages) {
    const stageName = stage.metadata?.name;
    if (stageName && stage.spec?.requestedFreight?.some((r) => r.origin?.name === warehouseName)) {
      reachableStages.add(stageName);
    }
  }
  return reachableStages;
};

export const getStagesForImage = (
  repoURL: string,
  warehouses: WarehouseExpanded[],
  stages: Stage[]
): Set<string> => {
  const warehousesForImage = findWarehousesForImageRepo(repoURL, warehouses);
  const allReachableStages = new Set<string>();
  for (const warehouseName of warehousesForImage) {
    for (const stageName of findStagesForWarehouse(warehouseName, stages)) {
      allReachableStages.add(stageName);
    }
  }
  return allReachableStages;
};

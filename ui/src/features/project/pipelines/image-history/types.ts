import { ImageStageMap } from '@ui/gen/api/service/v1alpha1/service_pb';

export type ProcessedTagMap = {
  tags: Record<string, ImageStageMap>;
};

export type ProcessedImages = Record<string, ProcessedTagMap>;

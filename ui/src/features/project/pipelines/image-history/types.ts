import { ImageStageMap } from '@ui/gen/api/v2/models';

export type ProcessedTagMap = {
  tags: Record<string, ImageStageMap>;
};

export type ProcessedImages = Record<string, ProcessedTagMap>;

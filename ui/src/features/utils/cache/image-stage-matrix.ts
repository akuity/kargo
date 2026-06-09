import { queryClient } from '@ui/config/query-client';
import { PromotionStatusPhase } from '@ui/features/common/promotion-status/utils';
import { getListImagesQueryKey, listImagesResponse } from '@ui/gen/api/v2/core/core';
import { ImageStageMap, Stage, TagMap } from '@ui/gen/api/v2/models';

export default {
  /**
   * problem: images stage matrix view is bit heavy calculation so we offload to the API
   *
   * this breaks the live view. For example stage "X" is promoted with "Y" image, the only way for now is to re-construct the matrix from scratch
   * even doing it in API is expensive
   *
   * there is simple definition of this matrix:
   *    the image row and stage columns box represents
   *    how far is that image from that stage
   *
   * in order to update the matrix, we just need the new stage promotion event as promotion event moves the images from stage to stage
   * if we get the last succeeded promotion, we just need to reset the new image distance from stage and bump other images
   *
   * this event is idempotent for the fact that if the image-dev distance is 0 then we don't want to change anything
   */
  update: (stage: Stage) => {
    // @ts-expect-error project name always there
    const projectName: string = stage?.metadata?.namespace;

    // @ts-expect-error stage name always there
    const stageName: string = stage?.metadata?.name;

    const lastPromotion = stage?.status?.lastPromotion;

    // if last promotion did not success then skip
    if ((lastPromotion?.status?.phase as PromotionStatusPhase) !== PromotionStatusPhase.SUCCEEDED) {
      return;
    }

    const cached = queryClient.getQueryData<listImagesResponse>(getListImagesQueryKey(projectName));

    const imageStageMatrix = cached?.data || {};

    const lastPromotionFreight = lastPromotion?.freight;

    if (!lastPromotionFreight) {
      // promotion doesn't succeed without freight but even if thats the case (or rather bug), matrix doesn't need update
      return;
    }

    const images = lastPromotionFreight?.images || [];

    if (images.length === 0) {
      // again if its not image related promotion then matrix doesn't need update
      return;
    }

    for (const image of images) {
      const repoURL: string = image.repoURL || '';

      const tag: string = image.tag || '';

      // check the existance in matrix
      if (!imageStageMatrix[repoURL]?.tags) {
        imageStageMatrix[repoURL] = { tags: {} } as TagMap;
      }

      if (!imageStageMatrix[repoURL].tags![tag]) {
        imageStageMatrix[repoURL].tags![tag] = { stages: {} } as ImageStageMap;
      }

      // idempotent check
      if (imageStageMatrix[repoURL].tags![tag].stages![stageName] === 0) {
        continue;
      }

      // bump all the tags<-><stageName> distance by 1 because promotion made them 1 step away
      for (const oldPromotedTag of Object.keys(imageStageMatrix[repoURL].tags!)) {
        const currentDistance =
          imageStageMatrix[repoURL].tags![oldPromotedTag]?.stages?.[stageName];

        if (currentDistance !== undefined && currentDistance >= 0) {
          imageStageMatrix[repoURL].tags![oldPromotedTag].stages![stageName] = currentDistance + 1;
        }
      }

      // reset distance for this tag<-><stageName>
      imageStageMatrix[repoURL].tags![tag].stages![stageName] = 0;
    }

    queryClient.setQueryData(
      getListImagesQueryKey(projectName),
      cached ? { ...cached, data: imageStageMatrix } : imageStageMatrix
    );
  }
};

import { DndContext } from '@dnd-kit/core';
import React, { useEffect, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useQueryFreightsRest } from '@ui/gen/api/v2/core/core';

import { useManualApprovalModal } from '../use-manual-approval-modal';

type Props = React.PropsWithChildren & {
  projectName: string;
};

export const DndPromotionContext = ({ children, projectName }: Props) => {
  const navigate = useNavigate();
  const [stage, setStage] = useState<string>();
  const [freight, setFreight] = useState<string>();

  const query = useQueryFreightsRest(
    projectName || '',
    { stage },
    { query: { enabled: !!stage && !!freight } }
  );

  const showManualApproveModal = useManualApprovalModal();

  useEffect(() => {
    // the query might be triggered from another place, so we need to watch stage and freight too
    if (query.data?.data && stage && freight) {
      const promotionEligibleFreight = query?.data?.data?.groups?.['']?.items || [];

      const promotionEligible = Boolean(
        promotionEligibleFreight?.find((i) => i?.metadata?.name === freight)
      );

      const navigateToPromotion = () =>
        navigate(
          generatePath(paths.promote, {
            name: projectName,
            freight: freight,
            stage: stage
          })
        );

      if (promotionEligible) {
        navigateToPromotion();
        setStage(undefined);
        setFreight(undefined);
      } else {
        showManualApproveModal({
          freight,
          stage,
          projectName,
          onClose: () => {
            setStage(undefined);
            setFreight(undefined);
          },
          onApprove: () => {
            navigate(
              generatePath(paths.promote, {
                name: projectName,
                freight: freight,
                stage
              })
            );
          }
        });
      }
    }
  }, [query.data?.data, stage, freight]);

  return (
    <DndContext
      autoScroll={false}
      onDragEnd={({ active, over }) => {
        // dropped outside of any stage -- nothing to do
        if (!over) {
          return;
        }

        const stageName = over.id as string;
        const originName = active?.data?.current?.originName;
        const requestedFreightNames: string[] = over.data?.current?.requestedFreightNames || [];

        // a closed promotion window forbids promotion of this Stage entirely --
        // the API would reject it at admission, so don't even start the flow. The
        // overlay already marked the Stage as frozen for the whole drag, so there
        // is nothing left to tell the user here
        if (over.data?.current?.promotionWindowClosed) {
          return;
        }

        // make sure that the freight can be promoted to this stage by checking
        // the origin. The overlay already told the user why it can't while
        // they hovered, so a rejected drop does nothing
        if (requestedFreightNames.includes(originName)) {
          setStage(stageName);
          setFreight(active.id as string);
        }
      }}
    >
      {children}
    </DndContext>
  );
};

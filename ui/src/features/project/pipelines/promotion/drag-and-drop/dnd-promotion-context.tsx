import { DndContext } from '@dnd-kit/core';
import { notification } from 'antd';
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

        // make sure that the freight can be promoted to this stage by checking the origin
        if (requestedFreightNames.includes(originName)) {
          setStage(stageName);
          setFreight(active.id as string);
          return;
        }

        notification.error({
          message: `Can't promote to Stage "${stageName}"`,
          description:
            `This Stage isn't connected to the "${originName}" Warehouse. ` +
            'Freight can only be promoted to Stages that request Freight from its Warehouse.',
          placement: 'bottomRight'
        });
      }}
    >
      {children}
    </DndContext>
  );
};

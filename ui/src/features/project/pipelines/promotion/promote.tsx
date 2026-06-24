import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Flex } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getCurrentFreightForComparison } from '@ui/features/common/utils';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { usePromoteDownstream, usePromoteToStage } from '@ui/gen/api/v2/core/core';
import { Freight, Stage } from '@ui/gen/api/v2/models';

import { useDictionaryContext } from '../context/dictionary-context';
import { isStageControlFlow } from '../nodes/stage-meta-utils';

import { FreightDetails } from './freight-details';
import styles from './promote.module.less';

type PromoteProps = ModalComponentProps & {
  stage: Stage;
  freight: Freight;
};

export const Promote = (props: PromoteProps) => {
  const actionContext = useActionContext();
  const navigate = useNavigate();
  const { promoteTabs } = useExtensionsContext();

  const dictionaryContext = useDictionaryContext();

  const isDownstreamPromotion =
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM || isStageControlFlow(props.stage);

  const freightAlias = props.freight?.alias;
  const stageName = props.stage?.metadata?.name;
  const projectName = props.stage?.metadata?.namespace;

  const currentFreightOnStage = useMemo(
    () => getCurrentFreightForComparison(props.stage, props.freight),
    [props.stage, props.freight]
  );

  const promoteActionMutation = usePromoteToStage({
    mutation: {
      onSuccess: (response) => {
        if (response.status !== 201) {
          return;
        }
        // navigate
        navigate(
          generatePath(paths.promotion, {
            name: projectName,
            promotionId: response.data?.metadata?.name
          })
        );

        actionContext?.cancel();
      }
    }
  });

  const promoteDownstreamActionMutation = usePromoteDownstream({
    mutation: {
      onSuccess: (response) => {
        if (response.status !== 201) {
          return;
        }
        // navigate
        navigate(
          generatePath(paths.project, {
            name: projectName
          })
        );

        actionContext?.cancel();
      }
    }
  });

  const onPromote = () => {
    const payload = {
      stage: stageName || '',
      project: projectName || '',
      data: { freight: props.freight?.metadata?.name }
    };

    if (isDownstreamPromotion) {
      promoteDownstreamActionMutation.mutate(payload);
      return;
    }

    promoteActionMutation.mutate(payload);
  };

  let promotingTo = stageName || '';

  if (isDownstreamPromotion) {
    promotingTo = [...(dictionaryContext?.subscribersByStage?.[promotingTo] || [])].join(', ');
  }

  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      title={
        <Flex align='center'>
          Promote {freightAlias} to {promotingTo}
        </Flex>
      }
      size='large'
      width={'1400px'}
      footer={
        <Flex vertical gap={12}>
          <Button
            size='large'
            className={classNames(styles['promote-btn'])}
            icon={<FontAwesomeIcon icon={faTruckArrowRight} />}
            onClick={onPromote}
            loading={promoteActionMutation.isPending || promoteDownstreamActionMutation.isPending}
          >
            {isDownstreamPromotion ? 'Promote to downstream' : 'Promote'}
          </Button>
        </Flex>
      }
    >
      <div className='-mt-6'>
        <FreightDetails
          freight={props.freight}
          comparison={{ currentFreight: currentFreightOnStage }}
          additionalTabs={promoteTabs.map((data, index) => ({
            children: <data.component freight={props.freight} stage={props.stage} />,
            key: String(data.label + index),
            label: data.label,
            icon: data.icon
          }))}
        />
      </div>
    </Drawer>
  );
};

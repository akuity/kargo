import { useMutation } from '@connectrpc/connect-query';
import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Flex } from 'antd';
import classNames from 'classnames';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import {
  promoteDownstream,
  promoteToStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightDetails } from './freight-details';
import styles from './promote.module.less';
import { PromotionGraph } from './promotion-graph';

type PromoteProps = ModalComponentProps & {
  stage: Stage;
  freight: Freight;
};

export const Promote = (props: PromoteProps) => {
  const actionContext = useActionContext();
  const navigate = useNavigate();

  const freightAlias = props.freight?.alias;
  const stageName = props.stage?.metadata?.name;
  const projectName = props.stage?.metadata?.namespace;

  const promoteActionMutation = useMutation(promoteToStage, {
    onSuccess: (response) => {
      // navigate
      navigate(
        generatePath(paths.promotion, {
          name: projectName,
          promotionId: response.promotion?.metadata?.name
        })
      );

      actionContext?.cancel();
    }
  });

  const promoteDownstreamActionMutation = useMutation(promoteDownstream, {
    onSuccess: () => {
      // navigate
      navigate(
        generatePath(paths.project, {
          name: projectName
        })
      );

      actionContext?.cancel();
    }
  });

  const onPromote = () => {
    const payload = {
      stage: stageName,
      project: projectName,
      freight: props.freight?.metadata?.name
    };

    if (actionContext?.action?.type === IAction.PROMOTE) {
      promoteActionMutation.mutate(payload);
    } else if (actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM) {
      promoteDownstreamActionMutation.mutate(payload);
    }
  };

  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      title={
        <Flex align='center'>
          Promote {freightAlias} to {stageName}
        </Flex>
      }
      size='large'
      width={'1224px'}
      footer={
        <Button
          size='large'
          className={classNames(styles['promote-btn'], 'ml-auto mt-5')}
          icon={<FontAwesomeIcon icon={faTruckArrowRight} />}
          onClick={onPromote}
          loading={promoteActionMutation.isPending || promoteDownstreamActionMutation.isPending}
        >
          Promote{actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM && ' to downstream'}
        </Button>
      }
    >
      <FreightDetails freight={props.freight} />

      <PromotionGraph freight={props.freight} stage={props.stage} />
    </Drawer>
  );
};

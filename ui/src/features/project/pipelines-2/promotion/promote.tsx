import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Flex } from 'antd';
import classNames from 'classnames';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightDetails } from './freight-details';
import styles from './promote.module.less';
import { PromotionGraph } from './promotion-graph';

type PromoteProps = ModalComponentProps & {
  stage: Stage;
  freight: Freight;
};

export const Promote = (props: PromoteProps) => {
  const freightAlias = props.freight?.alias;
  const stageName = props.stage?.metadata?.name;

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
        >
          Promote
        </Button>
      }
    >
      <FreightDetails freight={props.freight} />

      <PromotionGraph freight={props.freight} stage={props.stage} />
    </Drawer>
  );
};

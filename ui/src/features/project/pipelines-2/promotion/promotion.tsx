import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { ReactFlow } from '@xyflow/react';
import { Button, Descriptions, Drawer, Flex } from 'antd';
import classNames from 'classnames';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { FreightTable } from '@ui/features/project/pipelines-2/freight/freight-table';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useGetFreightCreation } from '../freight/use-get-freight-creation';

import { nodeTypes } from './mini-graph/constant';
import { useMiniPromotionGraph } from './mini-graph/use-mini-promotion-graph';
import styles from './promotion.module.less';

type PromotionProps = ModalComponentProps & {
  stage: Stage;
  freight: Freight;
};

export const Promotion = (props: PromotionProps) => {
  const freightAlias = props.freight?.alias;
  const stageName = props.stage?.metadata?.name;

  const freightCreatedAt = useGetFreightCreation(props.freight);

  const graph = useMiniPromotionGraph(props.stage, props.freight);

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
      <Descriptions
        column={2}
        size='small'
        bordered
        title='Freight'
        items={[
          {
            label: 'id',
            children: props.freight?.metadata?.name
          },
          {
            label: 'uid',
            children: props.freight?.metadata?.uid
          },
          {
            label: 'created',
            children: `${freightCreatedAt.relative}${freightCreatedAt.relative && ' , on'} ${freightCreatedAt.abs}`
          }
        ]}
      />

      <FreightTable className='mt-5' freight={props.freight} />

      <div className='bg-zinc-100 w-full h-[300px] rounded-lg'>
        <ReactFlow
          nodeTypes={nodeTypes}
          {...graph}
          fitView
          proOptions={{ hideAttribution: true }}
          panOnDrag={false}
          panOnScroll={false}
          nodesDraggable={false}
          elementsSelectable={false}
          maxZoom={1}
          minZoom={1}
        />
      </div>
    </Drawer>
  );
};

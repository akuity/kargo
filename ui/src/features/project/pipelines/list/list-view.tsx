import { Table } from 'antd';
import classNames from 'classnames';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useEventsWatcher } from '@ui/features/project/pipelines/graph/use-events-watcher';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { actionColumn } from './columns/action-column';
import { healthColumn } from './columns/health-column';
import { lastPromotionColumn } from './columns/last-promotion-column';
import { phaseColumn } from './columns/phase-column';
import { stageColumn } from './columns/stage-column';
import { versionColumn } from './columns/version-column';

type PipelineListViewProps = {
  stages: Stage[];
  warehouses: Warehouse[];
  className?: string;
  project: string;
};

export const PipelineListView = (props: PipelineListViewProps) => {
  const actionContext = useActionContext();

  useEventsWatcher(props.project);

  return (
    <div className={classNames(props.className, 'px-2')}>
      <Table
        dataSource={props.stages}
        rowKey={(stage) => `${stage?.metadata?.name}-${stage?.status?.observedGeneration}`}
        size='small'
        columns={[
          stageColumn(),
          phaseColumn(),
          healthColumn(),
          versionColumn(),
          lastPromotionColumn(),
          actionColumn({
            onPromote: (stage) => actionContext?.actPromote(IAction.PROMOTE, stage)
          })
        ]}
      />
    </div>
  );
};

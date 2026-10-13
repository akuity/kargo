import { faChartLine, faFileLines } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Modal, Tabs } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { stringify } from 'yaml';

import { useGetAnalysisRunRequest } from '@ui/gen/api/v2/verifications/verifications';

import YamlEditor from '../code-editor/yaml-editor-lazy';
import { ModalProps } from '../modal/use-modal';

import { aggregateTargets } from './aggregate-targets';
import { MetricLabel } from './metric-label/metric-label';
import { MetricPanel, SummaryPanel } from './panels';
import {
  analysisEndTime,
  analysisStatusLabel,
  analysisSubstatus,
  getAdjustedMetricPhase,
  transformMetrics
} from './transforms';
import { AnalysisStatus } from './types';

const cx = classNames;

interface AnalysisRunRequestModalProps {
  analysisName: string;
  images: string[];
}

export const AnalysisRunRequestModal = ({
  analysisName,
  images,
  visible,
  hide
}: AnalysisRunRequestModalProps & ModalProps) => {
  const [curTab, setCurTab] = useState('details');
  const { name: namespace } = useParams();

  const { data: analysisRunRequestData, isLoading } = useGetAnalysisRunRequest(
    namespace || '',
    analysisName || ''
  );

  const [request, status, transformedMetrics] = useMemo(() => {
    const request = analysisRunRequestData?.data;
    const [spec, status] = aggregateTargets(request);
    const transformedMetrics = transformMetrics(spec, status);

    return [request, status, transformedMetrics];
  }, [analysisRunRequestData, isLoading]);

  const tabItems = [
    {
      label: (
        <MetricLabel
          label='Summary'
          status={getAdjustedMetricPhase(status.phase as AnalysisStatus)}
          substatus={analysisSubstatus(status)}
        />
      ),
      key: 'analysis-summary',
      children: (
        <SummaryPanel
          title={analysisStatusLabel(status)}
          status={getAdjustedMetricPhase(status.phase as AnalysisStatus)}
          substatus={analysisSubstatus(status)}
          images={images}
          message={status.message}
          startTime={
            status.startedAt
              ? new Date(status.startedAt).getTime() / 1000
              : request?.metadata?.creationTimestamp
                ? new Date(request.metadata.creationTimestamp).getTime() / 1000
                : 0
          }
          endTime={
            status.completedAt
              ? new Date(status.completedAt).getTime() / 1000
              : analysisEndTime(status.metricResults ?? [])
          }
        />
      )
    },
    ...Object.values(transformedMetrics)
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((metric) => ({
        label: (
          <MetricLabel
            label={metric.name}
            status={metric.status.adjustedPhase}
            substatus={metric.status.substatus}
          />
        ),
        key: metric.name,
        children: (
          <MetricPanel
            metricName={metric.name}
            status={(metric.status.phase ?? AnalysisStatus.Unknown) as AnalysisStatus}
            substatus={metric.status.substatus}
            metricSpec={metric.spec}
            metricResults={metric.status}
          />
        )
      }))
  ];

  return (
    <Modal title={analysisName} width={866} footer={null} open={visible} onCancel={hide}>
      <Tabs onChange={(tab) => setCurTab(tab)} activeKey={curTab}>
        <Tabs.TabPane key='details' tab='Details' icon={<FontAwesomeIcon icon={faChartLine} />}>
          <Tabs
            className={cx('tabs')}
            items={tabItems}
            tabPosition='left'
            size='small'
            tabBarGutter={12}
          />
        </Tabs.TabPane>

        <Tabs.TabPane key='yaml' tab='YAML' icon={<FontAwesomeIcon icon={faFileLines} />}>
          <YamlEditor
            value={stringify(analysisRunRequestData?.data)}
            height='500px'
            isLoading={isLoading}
            disabled
          />
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};

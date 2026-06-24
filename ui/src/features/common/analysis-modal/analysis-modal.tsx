import { faChartLine, faFileLines, faHistory } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Modal, Tabs } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { stringify } from 'yaml';

import { useGetAnalysisRun } from '@ui/gen/api/v2/verifications/verifications';

import { AnalysisRunLogs } from '../analysis-run-logs/analysis-run-logs';
import YamlEditor from '../code-editor/yaml-editor-lazy';
import { ModalProps } from '../modal/use-modal';

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

interface AnalysisModalProps {
  analysisName: string;
  images: string[];
}

export const AnalysisModal = ({
  analysisName,
  images,
  visible,
  hide
}: AnalysisModalProps & ModalProps) => {
  const [curTab, setCurTab] = useState('details');
  const { name: namespace } = useParams();

  const { data: analysisRunData, isLoading } = useGetAnalysisRun(
    namespace || '',
    analysisName || ''
  );

  const [analysis, transformedMetrics] = useMemo(() => {
    const analysis = analysisRunData?.data;
    const transformedMetrics = transformMetrics(analysis?.spec, analysis?.status);

    return [analysis, transformedMetrics];
  }, [analysisRunData, isLoading]);

  const tabItems = [
    {
      label: (
        <MetricLabel
          label='Summary'
          status={getAdjustedMetricPhase(analysis?.status?.phase as AnalysisStatus)}
          substatus={analysisSubstatus(analysis?.status)}
        />
      ),
      key: 'analysis-summary',
      children: (
        <SummaryPanel
          title={analysisStatusLabel(analysis?.status)}
          status={getAdjustedMetricPhase(analysis?.status?.phase as AnalysisStatus)}
          substatus={analysisSubstatus(analysis?.status)}
          images={images}
          message={analysis?.status?.message}
          startTime={
            analysis?.metadata?.creationTimestamp
              ? new Date(analysis.metadata.creationTimestamp).getTime() / 1000
              : 0
          }
          endTime={analysisEndTime(analysis?.status?.metricResults ?? [])}
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
            value={stringify(analysisRunData?.data)}
            height='500px'
            isLoading={isLoading}
            disabled
          />
        </Tabs.TabPane>

        <Tabs.TabPane key='logs' tab='Logs' icon={<FontAwesomeIcon icon={faHistory} />}>
          <AnalysisRunLogs linkFullScreen analysisRun={analysisRunData?.data} />
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};

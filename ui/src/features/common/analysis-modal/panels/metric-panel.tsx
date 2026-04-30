// eslint-disable-file @typescript-eslint/ban-ts-comment
import { faChartLine, faList } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Radio, Typography } from 'antd';
import type { RadioChangeEvent } from 'antd';
import classNames from 'classnames';
import { useState } from 'react';

import {
  METRIC_CONSECUTIVE_ERROR_LIMIT_DEFAULT,
  METRIC_FAILURE_LIMIT_DEFAULT,
  METRIC_INCONCLUSIVE_LIMIT_DEFAULT
} from '../constants';
import { CriteriaList } from '../criteria-list/criteria-list';
import { Header } from '../header/header';
import { Legend } from '../legend/legend';
import { MetricChart } from '../metric-chart/metric-chart';
import { MetricTable } from '../metric-table/metric-table';
import { QueryBox } from '../query-box/query-box';
import { getFiniteNumber } from '../transforms';
import {
  AnalysisStatus,
  FunctionalStatus,
  TransformedMetricSpec,
  TransformedMetricStatus
} from '../types';

const { Paragraph, Title } = Typography;

interface MetricPanelProps {
  className?: string[] | string;
  metricName: string;
  metricSpec?: TransformedMetricSpec;
  metricResults: TransformedMetricStatus;
  status: AnalysisStatus;
  substatus?: FunctionalStatus.ERROR | FunctionalStatus.WARNING;
}

export const MetricPanel = ({
  className,
  metricName,
  metricSpec,
  metricResults,
  status,
  substatus
}: MetricPanelProps) => {
  const consecutiveErrorLimit = getFiniteNumber(
    metricSpec?.consecutiveErrorLimit,
    METRIC_CONSECUTIVE_ERROR_LIMIT_DEFAULT
  );
  const failureLimit = getFiniteNumber(metricSpec?.failureLimit, METRIC_FAILURE_LIMIT_DEFAULT);
  const inconclusiveLimit = getFiniteNumber(
    metricSpec?.inconclusiveLimit,
    METRIC_INCONCLUSIVE_LIMIT_DEFAULT
  );

  const canChartMetric = metricResults.chartable && metricResults.chartMax !== null;

  const [selectedView, setSelectedView] = useState(canChartMetric ? 'chart' : 'table');

  const onChangeView = ({ target: { value } }: RadioChangeEvent) => {
    setSelectedView(value);
  };

  return (
    <div className={classNames(className)}>
      <div className='w-full flex items-center my-2'>
        <Header
          title={metricName}
          subtitle={metricResults.statusLabel}
          status={metricResults.adjustedPhase}
          substatus={substatus}
          className='mr-auto'
        />
        {canChartMetric && (
          <Radio.Group onChange={onChangeView} value={selectedView} size='small'>
            <Radio.Button value='chart'>
              <FontAwesomeIcon icon={faChartLine} />
            </Radio.Button>
            <Radio.Button value='table'>
              <FontAwesomeIcon icon={faList} />
            </Radio.Button>
          </Radio.Group>
        )}
      </div>
      {status === AnalysisStatus.Pending && (
        <Paragraph style={{ marginTop: 12 }}>
          {metricName} analysis measurements have not yet begun. Measurement information will appear
          here when it becomes available.
        </Paragraph>
      )}
      {status !== AnalysisStatus.Pending && metricResults.transformedMeasurements.length === 0 && (
        <Paragraph style={{ marginTop: 12 }}>
          Measurement results for {metricName} cannot be displayed.
        </Paragraph>
      )}
      {status !== AnalysisStatus.Pending && metricResults.transformedMeasurements.length > 0 && (
        <>
          <Legend
            className='flex justify-end'
            errors={metricResults.error ?? 0}
            failures={metricResults.failed ?? 0}
            inconclusives={metricResults.inconclusive ?? 0}
            successes={metricResults.successful ?? 0}
          />
          {selectedView === 'chart' && (
            <MetricChart
              className={classNames('metric-section', 'mt-2')}
              data={metricResults.transformedMeasurements}
              max={metricResults?.chartMax}
              min={metricResults?.chartMin}
              failThresholds={metricSpec?.failThresholds || []}
              successThresholds={metricSpec?.successThresholds || []}
              yAxisLabel={metricResults?.name}
              conditionKeys={metricSpec?.conditionKeys || []}
            />
          )}
          {selectedView === 'table' && (
            <MetricTable
              className={classNames('metric-section', 'mt-2')}
              data={metricResults.transformedMeasurements}
              conditionKeys={metricSpec?.conditionKeys || []}
              failCondition={metricSpec?.failConditionLabel}
              successCondition={metricSpec?.successConditionLabel}
            />
          )}
        </>
      )}
      <div className={classNames('metric-section', 'mb-4')}>
        <Title className='mb-0' level={5}>
          Pass requirements
        </Title>
        <CriteriaList
          analysisStatus={status}
          maxConsecutiveErrors={consecutiveErrorLimit}
          maxFailures={failureLimit}
          maxInconclusives={inconclusiveLimit}
          consecutiveErrors={metricResults.consecutiveError ?? 0}
          failures={metricResults.failed ?? 0}
          inconclusives={metricResults.inconclusive ?? 0}
          showIcons={metricResults.measurements?.length > 0}
        />
      </div>
      {Array.isArray(metricSpec?.queries) && (metricSpec?.queries || []).length > 0 && (
        <>
          <div className={classNames('query-header')}>
            <Title className='mb-0' level={5}>
              {metricSpec.queries.length > 1 ? 'Queries' : 'Query'}
            </Title>
          </div>
          {metricSpec.queries.map((query: string) => (
            <QueryBox
              key={`query-box-${query}`}
              className={classNames('query-box')}
              query={query}
            />
          ))}
        </>
      )}
    </div>
  );
};

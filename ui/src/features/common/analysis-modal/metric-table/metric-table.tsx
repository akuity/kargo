import { Table, Typography } from 'antd';
import classNames from 'classnames';
import moment from 'moment';

import { StatusIndicator } from '../status-indicator/status-indicator';
import { AnalysisStatus, TransformedMeasurement, TransformedValueObject } from '../types';

import styles from './metric-table.module.less';

const { Column } = Table;
const { Text } = Typography;

const isObject = (tValue: TransformedValueObject | number | string | null) =>
  typeof tValue === 'object' && !Array.isArray(tValue) && tValue !== null;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const columnValueLabel = (value: any, valueKey: string) =>
  isObject(value) && valueKey in (value as TransformedValueObject)
    ? (value as TransformedValueObject)[valueKey]
    : '';

interface MetricTableProps {
  className?: string;
  conditionKeys: string[];
  data: TransformedMeasurement[];
  failCondition?: string;
  successCondition?: string;
}

export const MetricTable = ({
  className,
  conditionKeys,
  data,
  failCondition,
  successCondition
}: MetricTableProps) => (
  <div className={className}>
    <Table
      className={styles['metric-table']}
      dataSource={data}
      size='small'
      pagination={false}
      scroll={{ y: 190 }}
    >
      <Column
        key='status'
        dataIndex='phase'
        width={28}
        render={(phase: AnalysisStatus) => <StatusIndicator size='small' status={phase} />}
        align='center'
      />
      {conditionKeys.length > 0 ? (
        <>
          {conditionKeys.map((cKey) => (
            <Column
              key={cKey}
              title={`Data Point ${cKey}`}
              render={(columnValue: TransformedMeasurement) => {
                const isError = columnValue.phase === AnalysisStatus.Error;
                const errorMessage = columnValue.message ?? 'Measurement error';
                const label = isError
                  ? errorMessage
                  : columnValueLabel(columnValue.tableValue, cKey);
                return <span className={classNames(isError && 'italic')}>{label}</span>;
              }}
            />
          ))}
        </>
      ) : (
        <Column key='value' title='Value' dataIndex='tableValue' />
      )}
      <Column
        key='time'
        title='Time'
        render={(measurement: TransformedMeasurement) => (
          <span>{moment.unix(Number(measurement?.startedAt?.seconds)).format()}</span>
        )}
      />
    </Table>
    {failCondition !== null && (
      <Text className={classNames('condition', 'is-ERROR')} type='secondary'>
        Failure condition: {failCondition}
      </Text>
    )}
    {successCondition !== null && (
      <Text className={classNames('condition', 'is-SUCCESS')} type='secondary'>
        Success condition: {successCondition}
      </Text>
    )}
  </div>
);

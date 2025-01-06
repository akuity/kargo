// eslint-disable-file @typescript-eslint/ban-ts-comment
import { Typography } from 'antd';
import classNames from 'classnames';
import moment from 'moment';
import {
  CartesianGrid,
  DotProps,
  Label,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  TooltipProps,
  XAxis,
  YAxis
} from 'recharts';
import { NameType, ValueType } from 'recharts/types/component/DefaultTooltipContent';

import { timestampDate } from '@ui/utils/connectrpc-utils';

import { StatusIndicator } from '../status-indicator/status-indicator';
import { AnalysisStatus, TransformedMeasurement } from '../types';
import { chartDotColors } from '../utils';

import styles from './metric-chart.module.less';

const { Text } = Typography;

const CHART_HEIGHT = 254;
const X_AXIS_HEIGHT = 45;

const defaultValueFormatter = (value: number | string | null) =>
  value === null ? '' : value.toString();

const timeTickFormatter = (axisData?: string) => {
  if (axisData === undefined) {
    return '';
  }
  return moment(axisData).format('LT');
};

type MeasurementDotProps = DotProps & {
  payload?: {
    phase: AnalysisStatus;
    startedAt: string;
    value: string | null;
  };
};

const MeasurementDot = ({ cx, cy, payload }: MeasurementDotProps) => (
  <circle
    r={4}
    cx={cx}
    cy={cy ?? CHART_HEIGHT - X_AXIS_HEIGHT}
    className={chartDotColors(payload?.phase ?? AnalysisStatus.Unknown)}
  />
);

type TooltipContentProps = TooltipProps<ValueType, NameType> & {
  conditionKeys: string[];
  valueFormatter: (value: number | string | null) => string;
};

const TooltipContent = ({
  active,
  conditionKeys,
  payload,
  valueFormatter
}: TooltipContentProps) => {
  if (!active || payload?.length === 0 || !payload?.[0].payload) {
    return null;
  }

  const data = payload[0].payload;
  let label;
  if (data.phase === AnalysisStatus.Error) {
    label = data.message ?? 'Measurement error';
  } else if (conditionKeys.length > 0) {
    const sublabels = conditionKeys.map((cKey: string) =>
      conditionKeys.length > 1
        ? `${valueFormatter(data.chartValue[cKey])} (${cKey})`
        : valueFormatter(data.chartValue[cKey])
    );
    label = sublabels.join(' , ');
  } else {
    label = valueFormatter(data.chartValue);
  }

  return (
    <div className={styles['metric-chart-tooltip']}>
      <Text className='ml-4' type='secondary' style={{ fontSize: 12 }}>
        {moment(data.startedAt).format('LTS')}
      </Text>
      <div className={styles['metric-chart-tooltip-status']}>
        <StatusIndicator size='small' status={data.phase} />
        <Text>{label}</Text>
      </div>
    </div>
  );
};

interface MetricChartProps {
  className?: string;
  conditionKeys: string[];
  data: TransformedMeasurement[];
  failThresholds: number[];
  max?: number;
  min?: number;
  successThresholds: number[];
  valueFormatter?: (value: number | string | null) => string;
  yAxisFormatter?: (value: number | string, index: number) => string;
  yAxisLabel?: string;
}

export const MetricChart = ({
  className,
  conditionKeys,
  data,
  failThresholds,
  max,
  min,
  successThresholds,
  valueFormatter = defaultValueFormatter,
  yAxisFormatter = defaultValueFormatter,
  yAxisLabel
}: MetricChartProps) => {
  // show ticks at boundaries of analysis
  const startingTick = timestampDate(data[0]?.startedAt)?.toLocaleTimeString() ?? '';
  const endingTick = timestampDate(data[data.length - 1]?.finishedAt)?.toLocaleTimeString() ?? '';
  const timeTicks: (string | number)[] = [startingTick, endingTick];

  return (
    <ResponsiveContainer className={className} height={CHART_HEIGHT} width='100%'>
      <LineChart
        className={styles['metric-chart']}
        data={data}
        margin={{
          top: 0,
          right: 0,
          left: 0,
          bottom: 0
        }}
      >
        <CartesianGrid strokeDasharray='4 4' />
        <XAxis
          className={styles['chart-axis']}
          height={X_AXIS_HEIGHT}
          dataKey='startedAt'
          ticks={timeTicks}
          tickFormatter={timeTickFormatter}
        />
        <YAxis
          className={styles['chart-axis']}
          width={60}
          domain={[min ?? 0, max ?? 'auto']}
          tickFormatter={yAxisFormatter}
        >
          <Label
            className={styles['chart-label']}
            angle={-90}
            dx={-20}
            position='inside'
            value={yAxisLabel}
          />
        </YAxis>
        <Tooltip
          content={<TooltipContent conditionKeys={conditionKeys} valueFormatter={valueFormatter} />}
          filterNull={false}
          isAnimationActive={true}
        />
        {failThresholds !== null && (
          <>
            {failThresholds.map((threshold) => (
              <ReferenceLine
                key={`fail-line-${threshold}`}
                className={classNames(styles['reference-line'], styles['is-ERROR'])}
                y={threshold}
              />
            ))}
          </>
        )}
        {successThresholds !== null && (
          <>
            {successThresholds.map((threshold) => (
              <ReferenceLine
                key={`success-line-${threshold}`}
                className={classNames(styles['reference-line'], styles['is-SUCCESS'])}
                y={threshold}
              />
            ))}
          </>
        )}
        {conditionKeys.length === 0 ? (
          <Line
            className={styles['chart-line']}
            dataKey={conditionKeys.length === 0 ? 'chartValue' : `chartValue.${conditionKeys[0]}`}
            isAnimationActive={false}
            activeDot={false}
            dot={<MeasurementDot />}
          />
        ) : (
          <>
            {conditionKeys.map((cKey) => (
              <Line
                key={cKey}
                className={styles['chart-line']}
                dataKey={`chartValue.${cKey}`}
                isAnimationActive={false}
                activeDot={false}
                dot={<MeasurementDot />}
              />
            ))}
          </>
        )}
      </LineChart>
    </ResponsiveContainer>
  );
};

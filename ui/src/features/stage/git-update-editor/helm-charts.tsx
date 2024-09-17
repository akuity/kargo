import { faPlus, faTimesCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Col, Flex, Input, Row, Tag } from 'antd';
import { useState } from 'react';
import { FieldValues, useForm } from 'react-hook-form';

import { FieldContainer } from '../../common/form/field-container';

import styles from './git-update-editor.module.less';
import { OriginOverride } from './origin-override';
import { helmChartDependencyUpdateSchema } from './schemas';
import { ValuesTable } from './values-table';
import { WarehouseTooltip } from './warehouse-tooltip';

export const HelmCharts = () => {
  const { control, handleSubmit, reset } = useForm({
    defaultValues: {
      repo: '',
      name: '',
      warehouse: '',
      chartPath: ''
    },
    resolver: zodResolver(helmChartDependencyUpdateSchema)
  });

  const [charts, setCharts] = useState<
    { repo: string; name: string; warehouse?: string; chartPath?: string }[]
  >([]);

  const onAdd = (data: FieldValues) => {
    setCharts([
      ...charts,
      {
        repo: data.repo,
        name: data.name,
        warehouse: data.warehouse,
        chartPath: data.chartPath
      }
    ]);
    reset();
  };

  return (
    <Row gutter={8} className='mb-6'>
      <Col span={12}>
        <ValuesTable show={charts.length > 0} label='Charts'>
          {charts?.map((chart, i) => (
            <Flex
              key={`${chart.name}-${chart.repo}-${i}`}
              align='center'
              className={styles.imageItem}
            >
              {chart.name}
              <WarehouseTooltip warehouse={chart.warehouse} />
              <Flex align='center' className='ml-auto'>
                <Tag>
                  Repo: <b>{chart.repo}</b>
                </Tag>
                {chart.chartPath && (
                  <Tag>
                    Path: <b>{chart.chartPath}</b>
                  </Tag>
                )}
                <FontAwesomeIcon
                  icon={faTimesCircle}
                  className='cursor-pointer'
                  onClick={() => {
                    setCharts(charts.filter((_, index) => index !== i));
                  }}
                />
              </Flex>
            </Flex>
          ))}
        </ValuesTable>
      </Col>
      <Col span={12}>
        <div className={styles.form}>
          <FieldContainer
            label='Chart Name'
            name='name'
            control={control}
            required
            formItemClassName='mb-2'
          >
            {({ field }) => (
              <Input {...field} placeholder='my-chart' value={field.value as string} />
            )}
          </FieldContainer>

          <OriginOverride>
            <FieldContainer label='Warehouse Origin' name='warehouse' control={control}>
              {({ field }) => (
                <Input {...field} placeholder='my-warehouse' value={field.value as string} />
              )}
            </FieldContainer>
          </OriginOverride>

          <FieldContainer
            label='Repository'
            name='repo'
            control={control}
            required
            formItemClassName='mb-4'
          >
            {({ field }) => (
              <Input {...field} placeholder='my-repository' value={field.value as string} />
            )}
          </FieldContainer>

          <FieldContainer
            label='Chart Path'
            name='chartPath'
            control={control}
            formItemClassName='mb-4'
          >
            {({ field }) => (
              <Input {...field} placeholder='path/to/chart' value={field.value as string} />
            )}
          </FieldContainer>

          <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={handleSubmit(onAdd)}>
            Add Chart
          </Button>
        </div>
      </Col>
    </Row>
  );
};

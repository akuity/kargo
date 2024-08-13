import { faPlus, faTimesCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Col, Flex, Input, Row, Tag } from 'antd';
import { useState } from 'react';
import { FieldValues, useForm } from 'react-hook-form';

import { FieldContainer } from '../../common/form/field-container';

import styles from './git-update-editor.module.less';
import { OriginOverride } from './origin-override';
import { helmImageUpdateSchema } from './schemas';
import { ValuesTable } from './values-table';
import { WarehouseTooltip } from './warehouse-tooltip';

export const HelmImages = () => {
  const { control, handleSubmit, reset } = useForm({
    defaultValues: {
      image: '',
      warehouse: '',
      valuesFilePath: '',
      key: '',
      value: ''
    },
    resolver: zodResolver(helmImageUpdateSchema)
  });

  const [images, setImages] = useState<
    { image: string; warehouse?: string; valuesFilePath: string; key: string; value: string }[]
  >([]);

  const onAdd = (data: FieldValues) => {
    setImages([
      ...images,
      {
        image: data.image,
        warehouse: data.warehouse,
        valuesFilePath: data.valuesFilePath,
        key: data.key,
        value: data.value
      }
    ]);
    reset();
  };

  return (
    <Row gutter={8} className='mb-6'>
      <Col span={12}>
        <ValuesTable show={images.length > 0}>
          {images?.map((img, i) => (
            <Flex key={`${img.image}-${i}`} align='center' className={styles.imageItem}>
              {img.image}
              <WarehouseTooltip warehouse={img.warehouse} />
              <Flex align='center' className='ml-auto'>
                <Tag>
                  Path: <b>{img.valuesFilePath}</b>
                </Tag>
                <Tag>
                  Key: <b>{img.key}</b>
                </Tag>
                <Tag>
                  Value: <b>{img.value}</b>
                </Tag>
                <FontAwesomeIcon
                  icon={faTimesCircle}
                  className='cursor-pointer'
                  onClick={() => {
                    setImages(images.filter((_, index) => index !== i));
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
            label='Image'
            name='image'
            control={control}
            required
            formItemClassName='mb-2'
          >
            {({ field }) => (
              <Input {...field} placeholder='my-image' value={field.value as string} />
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
            label='Values File Path'
            name='valuesFilePath'
            control={control}
            required
            formItemClassName='mb-4'
          >
            {({ field }) => (
              <Input {...field} placeholder='path/to/values/file' value={field.value as string} />
            )}
          </FieldContainer>

          <FieldContainer
            label='Key'
            name='key'
            control={control}
            required
            formItemClassName='mb-4'
          >
            {({ field }) => <Input {...field} placeholder='mykey' value={field.value as string} />}
          </FieldContainer>

          <FieldContainer
            label='Value'
            name='value'
            control={control}
            required
            formItemClassName='mb-4'
          >
            {({ field }) => (
              <Input {...field} placeholder='myvalue' value={field.value as string} />
            )}
          </FieldContainer>

          <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={handleSubmit(onAdd)}>
            Add Image
          </Button>
        </div>
      </Col>
    </Row>
  );
};

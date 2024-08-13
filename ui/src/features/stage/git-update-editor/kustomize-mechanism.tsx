import { faPlus, faTimesCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Col, Flex, Input, Row, Switch, Tag } from 'antd';
import { useState } from 'react';
import { FieldValues, useForm } from 'react-hook-form';

import { FieldContainer } from '../../common/form/field-container';

import styles from './git-update-editor.module.less';
import { OriginOverride } from './origin-override';
import { kustomizeImageUpdateSchema } from './schemas';
import { UseDigestTag } from './use-digest-tag';
import { ValuesTable } from './values-table';
import { WarehouseTooltip } from './warehouse-tooltip';

export const KustomizeMechanism = () => {
  const { control, handleSubmit, reset } = useForm({
    defaultValues: {
      useDigest: false,
      image: '',
      path: '',
      warehouse: ''
    },
    resolver: zodResolver(kustomizeImageUpdateSchema)
  });

  const [images, setImages] = useState<
    { image: string; useDigest?: boolean; warehouse?: string; path: string }[]
  >([]);

  const onAdd = (data: FieldValues) => {
    setImages([
      ...images,
      {
        image: data.image,
        useDigest: data.useDigest,
        warehouse: data.warehouse,
        path: data.path
      }
    ]);
    reset();
  };

  return (
    <Row gutter={8} className='mb-6'>
      <Col span={12}>
        <ValuesTable show={images?.length > 0}>
          {images?.map((img, i) => (
            <Flex key={`${img.image}-${i}`} align='center' className={styles.imageItem}>
              {img.image}
              <WarehouseTooltip warehouse={img.warehouse} />
              <Flex align='center' className='ml-auto'>
                <UseDigestTag visible={img.useDigest} />
                <Tag>
                  Path: <b>{img.path}</b>
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
          <Flex gap={8} align='end'>
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
            <FieldContainer name='useDigest' control={control} formItemClassName='mb-2'>
              {({ field }) => (
                <Flex align='center'>
                  <Switch
                    value={!!field.value as boolean}
                    checked={field.value as boolean}
                    onChange={(value) => field.onChange(value)}
                  />
                  <div className='ml-2'>Use Digest</div>
                </Flex>
              )}
            </FieldContainer>
          </Flex>
          <OriginOverride>
            <FieldContainer label='Warehouse Origin' name='warehouse' control={control}>
              {({ field }) => (
                <Input {...field} placeholder='my-warehouse' value={field.value as string} />
              )}
            </FieldContainer>
          </OriginOverride>
          <FieldContainer
            label='Path'
            name='path'
            control={control}
            required
            formItemClassName='mb-6'
          >
            {({ field }) => <Input {...field} placeholder='path' value={field.value as string} />}
          </FieldContainer>
          <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={handleSubmit(onAdd)}>
            Add Image
          </Button>
        </div>
      </Col>
    </Row>
  );
};

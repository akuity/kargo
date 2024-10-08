import { faInfoCircle, faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Alert, AutoComplete, Button, Flex, Select, Switch, Tooltip } from 'antd';
import classNames from 'classnames';
import { useForm } from 'react-hook-form';

import { FreightOrigin, FreightRequest, FreightSources } from '@ui/gen/v1alpha1/generated_pb';

import { FieldContainer } from '../common/form/field-container';

import { requestedFreightSchema } from './schemas';

export const RequestedFreightEditor = ({
  onSubmit,
  warehouses,
  stages
}: {
  onSubmit: (data: FreightRequest) => void;
  warehouses?: string[];
  stages?: string[];
}) => {
  const { control, handleSubmit, watch, reset } = useForm({
    defaultValues: {
      warehouse: '',
      sources: { direct: true, upstreamStages: [] }
    },
    resolver: zodResolver(requestedFreightSchema)
  });

  const direct = watch('sources.direct');

  return (
    <div className='w-full rounded-md bg-gray-50 p-3 mb-6'>
      {(warehouses?.length || 0) <= 0 && (
        <Alert
          message='No Warehouses exist for this project. To avoid errors, create a Warehouse before creating a Stage.'
          type='warning'
          showIcon
          className='mb-2'
        />
      )}
      <FieldContainer label='Warehouse' name='warehouse' control={control}>
        {({ field }) => (
          <AutoComplete
            options={warehouses?.map((name) => ({ value: name }))}
            {...field}
            placeholder='my-warehouse'
          />
        )}
      </FieldContainer>

      <FieldContainer name='sources.direct' control={control} formItemClassName='mb-3'>
        {({ field }) => (
          <Flex align='center'>
            <Switch checked={field.value as boolean} onChange={(value) => field.onChange(value)} />
            <div className='ml-2 font-semibold'>DIRECT</div>
            <Tooltip
              placement='right'
              title={
                <>
                  Turn <b>on</b> to source Freight from the Warehouse directly. <br />
                  <br /> Turn <b>off</b> to source Freight from upstream Stages.
                </>
              }
            >
              <FontAwesomeIcon icon={faInfoCircle} className='ml-2' />
            </Tooltip>
          </Flex>
        )}
      </FieldContainer>
      <div
        className={classNames({
          'opacity-50 cursor-not-allowed': direct
        })}
      >
        {!direct && (stages || []).length === 0 && (
          <Alert
            type='warning'
            className='mb-4'
            message='There are no other Stages in this Project. Before configuring Upstream Stages, create another Stage to avoid errors.'
          />
        )}

        <FieldContainer
          label='Upstream Stages'
          name='sources.upstreamStages'
          control={control}
          formItemClassName='mb-2'
        >
          {({ field }) => (
            <Select
              mode='multiple'
              placeholder='my-stage'
              options={(stages || []).map((name) => ({ value: name }))}
              value={field.value}
              disabled={direct}
              onChange={(value) => field.onChange(value)}
            />
          )}
        </FieldContainer>
      </div>

      <Button
        onClick={handleSubmit((value) => {
          onSubmit({
            origin: {
              kind: 'Warehouse',
              name: value.warehouse
            } as FreightOrigin,
            sources: {
              direct: value.sources.direct,
              stages: value.sources.upstreamStages
            } as Partial<FreightSources>
          } as FreightRequest);
          reset();
        })}
        icon={<FontAwesomeIcon icon={faPlus} />}
        className='mt-4'
      >
        Add Freight Request
      </Button>
    </div>
  );
};

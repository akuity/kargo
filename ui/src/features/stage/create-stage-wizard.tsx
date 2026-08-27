import { faTimes } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Col, Flex, Input, Row, Select, Typography } from 'antd';

import { FreightRequest, Stage } from '@ui/gen/api/v2/models';

import { PromotionStepsWizard } from './promotion-steps-wizard/promotion-steps-wizard';
import { RunnerWithConfiguration } from './promotion-steps-wizard/types';
import { RequestedFreight } from './requested-freight';
import { RequestedFreightEditor } from './requested-freight-editor';
import { ColorMapHex } from './utils';

// The controlled state of the Create Stage form. Shared so both the Create
// Stage drawer and the project-creation wizard render the exact same form.
export type StageWizardState = {
  name: string;
  color?: string;
  requestedFreight: FreightRequest[];
  steps: RunnerWithConfiguration[];
};

type CreateStageWizardProps = {
  value: StageWizardState;
  onChange: (next: StageWizardState) => void;
  projectName?: string;
  warehouses?: string[];
  stages?: Stage[];
};

export const CreateStageWizard = ({
  value,
  onChange,
  projectName,
  warehouses,
  stages
}: CreateStageWizardProps) => {
  const patch = (p: Partial<StageWizardState>) => onChange({ ...value, ...p });

  return (
    <>
      <Typography.Text className='block mb-1'>Name</Typography.Text>
      <Input
        className='mb-4'
        placeholder='my-stage'
        value={value.name}
        onChange={(e) => patch({ name: e.target.value })}
      />

      <Typography.Text className='block mb-1'>Color</Typography.Text>
      <Flex className='w-full mb-4' wrap>
        <Select
          placeholder='Select a color (optional)'
          className='w-full shrink-0'
          value={value.color}
          onChange={(color) => patch({ color })}
          options={Object.keys(ColorMapHex).map((color) => ({
            value: color,
            label: (
              <Flex align='center'>
                <div
                  className='mr-2 rounded'
                  style={{ backgroundColor: ColorMapHex[color], width: '10px', height: '10px' }}
                />
                {color.charAt(0).toUpperCase() + color.slice(1)}
              </Flex>
            )
          }))}
        />
        {value.color && (
          <Button
            onClick={() => patch({ color: undefined })}
            size='small'
            danger
            className='mt-2 ml-auto'
            icon={<FontAwesomeIcon icon={faTimes} />}
          >
            Clear Color
          </Button>
        )}
      </Flex>

      <Typography.Title level={4}>Requested Freight</Typography.Title>
      <Row className='mb-6' gutter={12}>
        <Col span={12}>
          {value.requestedFreight?.length > 0 ? (
            <RequestedFreight
              requestedFreight={value.requestedFreight}
              projectName={projectName}
              className='mb-4'
              itemStyle={{ width: '45%' }}
              onDelete={(index) =>
                patch({
                  requestedFreight: value.requestedFreight.filter((_, i) => i !== index)
                })
              }
              hideTitle
            />
          ) : (
            <Flex
              className='w-full h-full rounded-md bg-gray-50 dark:bg-neutral-800 text-gray-400 dark:text-neutral-500 font-medium text-center'
              align='center'
              justify='center'
            >
              Requested Freight are required to create a Stage.
              <br />
              Add a Freight Request using the form to the right to continue.
            </Flex>
          )}
        </Col>
        <Col span={12}>
          <RequestedFreightEditor
            warehouses={warehouses}
            stages={stages}
            onSubmit={(freight) =>
              patch({ requestedFreight: [...value.requestedFreight, freight] })
            }
          />
        </Col>
      </Row>

      <Typography.Title level={4}>Promotion Steps</Typography.Title>
      <PromotionStepsWizard steps={value.steps} onChange={(steps) => patch({ steps })} />
    </>
  );
};

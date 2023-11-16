import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Button, Modal, Space, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import {
  listStages,
  updateStage
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { prepareStageToEdit, prepareStageToSave } from './utils/edit-stage-utils';
import { getStageYAMLExample } from './utils/stage-yaml-example';

type Props = ModalComponentProps & {
  projectName: string;
  stageName: string;
};

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const EditStageModal = ({ visible, hide, projectName, stageName }: Props) => {
  const { data } = useQuery(listStages.useQuery({ project: projectName }));
  const stage = data?.stages.find((item) => item.metadata?.name === stageName);

  const { mutateAsync, isPending } = useMutation({
    ...updateStage.useMutation(),
    onSuccess: () => hide()
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: prepareStageToEdit(stage)
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    await mutateAsync({
      stage: {
        case: 'yaml',
        value: prepareStageToSave(stage, data.value)
      }
    });
  });

  return (
    <Modal
      destroyOnClose
      open={visible}
      title='Edit Stage'
      closable={false}
      width={680}
      footer={
        <div className='flex items-center justify-between'>
          <Typography.Link
            href='https://kargo.akuity.io/quickstart/#the-test-stage'
            target='_blank'
          >
            Documentation
          </Typography.Link>
          <Space>
            <Button onClick={hide}>Cancel</Button>
            <Button type='primary' onClick={onSubmit} loading={isPending}>
              Update
            </Button>
          </Space>
        </div>
      }
    >
      <FieldContainer label='YAML' name='value' control={control}>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            schema={schema as JSONSchema4}
            placeholder={getStageYAMLExample(projectName)}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};

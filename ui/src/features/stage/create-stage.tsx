import { faCode, faListCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Drawer, Tabs, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { generatePath, useNavigate } from 'react-router-dom';
import yaml, { parse, stringify } from 'yaml';
import { z } from 'zod';

import { paths } from '@ui/config/paths';
import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { FreightRequest, PromotionStep, Stage } from '@ui/gen/api/v2/models';
import { useCreateResource } from '@ui/gen/api/v2/resources/resources';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import { cleanEmptyObjectValues } from '@ui/utils/helpers';
import { zodValidators } from '@ui/utils/validators';

import { CreateStageWizard, StageWizardState } from './create-stage-wizard';
import { getStageYAMLExample } from './get-stage-yaml-example';
import { usePromotionWizardStepsState } from './promotion-steps-wizard/use-promotion-wizard-steps-state';

const formSchema = z.object({
  value: zodValidators.requiredString
});

const stageFormToYAML = (
  data: { name: string; color?: string; requestedFreight: FreightRequest[] },
  namespace: string,
  promotionTemplateSteps: PromotionStep[]
) => {
  return yaml.stringify({
    kind: 'Stage',
    apiVersion: 'kargo.akuity.io/v1alpha1',
    metadata: {
      name: data.name,
      namespace,
      ...(data.color &&
        data.color !== '' && {
          annotations: {
            'kargo.akuity.io/color': data.color
          }
        })
    },
    spec: {
      requestedFreight: data.requestedFreight,
      ...(promotionTemplateSteps?.length > 0 && {
        // IMPORTANT TO CLEANUP EMPTY VALUES OR UNEXPECTED CONFIG FOR PROMOTION STEP WOULD HAPPEN
        promotionTemplate: { spec: cleanEmptyObjectValues({ steps: promotionTemplateSteps }) }
      })
    }
  });
};

export const CreateStage = ({
  project,
  warehouses,
  stages
}: {
  project?: string;
  warehouses?: string[];
  stages?: Stage[];
}) => {
  const navigate = useNavigate();
  const close = () => navigate(generatePath(paths.project, { name: project }));
  const [tab, setTab] = useState('wizard');

  const { mutateAsync, isPending } = useCreateResource({ mutation: { onSuccess: () => close() } });

  const { control, handleSubmit, setValue, getValues } = useForm({
    defaultValues: {
      value: getStageYAMLExample(project || '')
    },
    resolver: zodResolver(formSchema)
  });

  const [wizardForm, setWizardForm] = useState<Omit<StageWizardState, 'steps'>>({
    name: '',
    requestedFreight: []
  });
  const promotionWizardStepsState = usePromotionWizardStepsState();

  const promotionSteps = (): PromotionStep[] =>
    (promotionWizardStepsState.state ?? []).map((step) => ({
      uses: step?.identifier,
      as: step?.as || '',
      if: '',
      continueOnError: step?.continueOnError || false,
      // step.state is type 'object' and it is safe to fake JSON type because it
      // doesn't matter for stageFormToYAML
      config: step?.state,
      vars: []
    }));

  const onSubmit = handleSubmit(async (data) => {
    let value = data.value;
    if (tab === 'wizard') {
      const unmarshalled = stageFormToYAML(wizardForm, project || '', promotionSteps());
      setValue('value', unmarshalled);
      value = unmarshalled;
    }
    await mutateAsync({ data: value });
  });

  if (!project) {
    return null;
  }

  return (
    <Drawer
      open={!!project}
      width={'80%'}
      onClose={close}
      title='Create Stage'
      extra={
        <Typography.Link
          href='https://docs.kargo.io/user-guide/how-to-guides/working-with-stages'
          target='_blank'
          className='ml-3'
        >
          Docs
        </Typography.Link>
      }
    >
      <Tabs
        className='-mt-4'
        onChange={(newTab) => {
          if (tab === 'wizard' && newTab === 'yaml') {
            setValue('value', stageFormToYAML(wizardForm, project || '', promotionSteps()));
          } else {
            const yaml = getValues('value');

            try {
              const stageSpec: Stage = parse(yaml);

              promotionWizardStepsState.setYAML(
                stringify(stageSpec?.spec?.promotionTemplate?.spec?.steps)
              );
            } catch (e) {
              // explicitly ignore
            }
          }
          setTab(newTab);
        }}
      >
        <Tabs.TabPane
          key='wizard'
          tab='Form'
          icon={<FontAwesomeIcon icon={faListCheck} />}
          className='mb-4'
        >
          <CreateStageWizard
            value={{ ...wizardForm, steps: promotionWizardStepsState.state }}
            onChange={(next) => {
              setWizardForm({
                name: next.name,
                color: next.color,
                requestedFreight: next.requestedFreight
              });
              promotionWizardStepsState.onChange(next.steps);
            }}
            projectName={project}
            warehouses={warehouses}
            stages={stages}
          />
        </Tabs.TabPane>
        <Tabs.TabPane key='yaml' tab='YAML' icon={<FontAwesomeIcon icon={faCode} />}>
          <FieldContainer name='value' control={control}>
            {({ field: { value, onChange } }) => (
              <YamlEditor
                value={value}
                onChange={(e) => onChange(e || '')}
                height='500px'
                schema={schema as JSONSchema4}
                placeholder={getStageYAMLExample(project)}
                resourceType='stages'
              />
            )}
          </FieldContainer>
        </Tabs.TabPane>
      </Tabs>
      <Button onClick={onSubmit} loading={isPending} type='primary'>
        Create
      </Button>
    </Drawer>
  );
};

export default CreateStage;

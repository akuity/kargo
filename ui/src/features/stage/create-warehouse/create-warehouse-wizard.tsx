import Form from '@rjsf/antd';
import validator from '@rjsf/validator-ajv8';
import { Checkbox, Input } from 'antd';
import AntdFormLabel from 'antd/es/form/FormItemLabel';
import { JSONSchema7 } from 'json-schema';
import { useState } from 'react';

import { RjsfConfigContext } from '@ui/features/common/form/rjsf/context';
import { DescriptionFieldTemplate } from '@ui/features/common/form/rjsf/description-field-template';
import { FieldTemplate } from '@ui/features/common/form/rjsf/field-template';
import { ObjectFieldTemplate } from '@ui/features/common/form/rjsf/object-field-template';
import rjsfStylesOverride from '@ui/features/common/form/rjsf/style-overrides.module.less';
import { WarehouseSpec } from '@ui/gen/v1alpha1/generated_pb';

import { warehouseCreateFormJSONSchema } from './schema';
import { SubscriptionWizard } from './subscription-wizard';

const warehouseCreateFormJSONSchemaWithoutSubscription = structuredClone(
  warehouseCreateFormJSONSchema
);
// @ts-expect-error check schema.ts
delete warehouseCreateFormJSONSchemaWithoutSubscription.properties.subscriptions;

type CreateWarehouseWizardProps = {
  // rjsf dynamic forms from JSON schema, its fine if it is not strongly typed
  formState: Record<string, unknown>;
  setFormState(nextFormState: object): void;
};

export const CreateWarehouseWizard = (props: CreateWarehouseWizardProps) => {
  const [showDescription, setShowDescription] = useState(false);

  const { formState, setFormState } = props;

  const subscriptions = (formState?.spec as WarehouseSpec)?.subscriptions || [];

  return (
    <RjsfConfigContext.Provider
      value={{
        showDescription,
        setConfig(nextConfig) {
          setShowDescription(nextConfig.showDescription);
        }
      }}
    >
      <div className='flex'>
        <Checkbox
          className='text-xs mb-2 ml-auto'
          checked={showDescription}
          onChange={(e) => setShowDescription(e.target.checked)}
        >
          Show description
        </Checkbox>
      </div>
      <div className={rjsfStylesOverride.container}>
        <AntdFormLabel prefixCls='' label='Name' htmlFor='warehouse-name' />
        <Input
          aria-label='warehouse-name'
          value={(formState?.name as string) || ''}
          onChange={(e) => setFormState({ ...formState, name: e.target.value })}
          className='mb-4 mt-2'
        />
        <Form
          key={`${showDescription}`}
          formData={formState?.spec || {}}
          onChange={({ formData }) =>
            setFormState({
              ...formState,
              spec: {
                ...(formState?.spec || {}),
                ...formData
              }
            })
          }
          schema={warehouseCreateFormJSONSchemaWithoutSubscription as JSONSchema7}
          validator={validator}
          templates={{
            ObjectFieldTemplate,
            DescriptionFieldTemplate,
            ArrayFieldDescriptionTemplate: () => null,
            FieldTemplate
          }}
          uiSchema={{
            'ui:submitButtonOptions': {
              norender: true
            }
          }}
        />

        <SubscriptionWizard
          subscriptions={subscriptions}
          onChange={(subscriptions) =>
            setFormState({
              ...formState,
              spec: {
                ...(formState?.spec || {}),
                subscriptions
              }
            })
          }
        />
      </div>
    </RjsfConfigContext.Provider>
  );
};

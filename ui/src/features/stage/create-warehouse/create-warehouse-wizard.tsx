import Form from '@rjsf/antd';
import { getDefaultRegistry } from '@rjsf/core';
import { RegistryFieldsType } from '@rjsf/utils';
import validator from '@rjsf/validator-ajv8';
import { Checkbox, Input } from 'antd';
import AntdFormLabel from 'antd/es/form/FormItemLabel';
import { JSONSchema7 } from 'json-schema';
import { useState } from 'react';

import { DescriptionFieldTemplate } from '@ui/features/common/form/rjsf/description-field-template';
import { FieldTemplate } from '@ui/features/common/form/rjsf/field-template';
import { ObjectFieldTemplate } from '@ui/features/common/form/rjsf/object-field-template';
import rjsfStylesOverride from '@ui/features/common/form/rjsf/style-overrides.module.less';

import { warehouseCreateFormJSONSchema } from './schema';

type CreateWarehouseWizardProps = {
  // rjsf dynamic forms from JSON schema, its fine if it is not strongly typed
  formState: Record<string, unknown>;
  setFormState(nextFormState: object): void;
};

export const CreateWarehouseWizard = (props: CreateWarehouseWizardProps) => {
  const [showDescription, setShowDescription] = useState(false);

  const { formState, setFormState } = props;

  return (
    <>
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
          onChange={({ formData }) => setFormState({ ...formState, spec: formData })}
          schema={warehouseCreateFormJSONSchema as JSONSchema7}
          validator={validator}
          templates={{
            ObjectFieldTemplate,
            DescriptionFieldTemplate: showDescription ? DescriptionFieldTemplate : () => null,
            ArrayFieldDescriptionTemplate: () => null,
            FieldTemplate
          }}
          fields={fields}
          uiSchema={{
            'ui:submitButtonOptions': {
              norender: true
            }
          }}
        />
      </div>
    </>
  );
};

const {
  fields: { ObjectField }
} = getDefaultRegistry();

const fields: RegistryFieldsType = {
  ObjectField: (props) => {
    console.log(props);
    return <ObjectField {...props} />;
  }
};

import { faFilter, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Input, Select, Typography } from 'antd';
import { useState } from 'react';
import { useController, useFieldArray, useFormContext, useWatch } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';

import { PromotionWindowFormValues } from './promotion-window-form-utils';
import {
  expressionOperators,
  selectorFromValues,
  selectorValues,
  valuelessOperators
} from './promotion-window-selector-utils';

const operatorOptions = expressionOperators.map((operator) => ({
  label: operator,
  value: operator
}));

const nameModeHelp = {
  exact: 'Matches one name, character for character.',
  glob: 'Shell-style wildcards, e.g. prod-*',
  regex: 'RE2 regular expression, unanchored by default.'
};

const subjectLabel = { stage: 'Stage', project: 'Project' };

type SelectorFieldsProps = {
  subject: 'stage' | 'project';
};

export const SelectorFields = ({ subject }: SelectorFieldsProps) => {
  const { control, getValues, setValue } = useFormContext<PromotionWindowFormValues>();

  const noun = subjectLabel[subject];

  const [restricting, setRestricting] = useState(() => !!selectorFromValues(getValues(subject)));

  const applyToEvery = () => {
    setValue(subject, selectorValues(), { shouldDirty: true });
    setRestricting(false);
  };

  const nameModeField = useController({ control, name: `${subject}.nameMode` }).field;
  const nameMode = nameModeField.value;

  const labels = useFieldArray({ control, name: `${subject}.labels` });

  const expressions = useFieldArray({ control, name: `${subject}.matchExpressions` });
  const expressionValues = useWatch({ control, name: `${subject}.matchExpressions` });

  if (!restricting) {
    return (
      <>
        <Typography.Text strong>{noun}s this window applies to</Typography.Text>
        <Typography.Paragraph type='secondary' className='text-xs !mt-1 !mb-3'>
          Every {noun}.
        </Typography.Paragraph>
        <Button
          size='small'
          type='dashed'
          className='!h-auto !py-1'
          icon={<FontAwesomeIcon icon={faFilter} size='sm' />}
          onClick={() => setRestricting(true)}
        >
          Restrict to specific {noun}s
        </Button>
      </>
    );
  }

  return (
    <>
      <Typography.Text strong>{noun}s this window applies to</Typography.Text>
      <Typography.Paragraph type='secondary' className='text-xs !mt-1 !mb-4'>
        Constraints are ANDed: a {noun} must satisfy the name and every label and expression to be
        covered by this window.
      </Typography.Paragraph>

      <FieldContainer
        control={control}
        name={`${subject}.name`}
        label='Name'
        description={nameModeHelp[nameMode]}
      >
        {({ field }) => (
          <Input
            {...field}
            placeholder={nameMode === 'exact' ? 'prod-us' : 'prod-*'}
            addonBefore={
              <Select
                {...nameModeField}
                style={{ width: 96 }}
                options={[
                  { label: 'Exact', value: 'exact' },
                  { label: 'glob:', value: 'glob' },
                  { label: 'regex:', value: 'regex' }
                ]}
              />
            }
          />
        )}
      </FieldContainer>

      <Typography.Text className='text-sm'>Labels</Typography.Text>
      <Flex vertical gap={8} className='mt-2 mb-6'>
        {labels.fields.map((label, index) => (
          <Flex key={label.id} gap={8} align='center'>
            <FieldContainer
              control={control}
              name={`${subject}.labels.${index}.key`}
              className='flex-1'
              formItemClassName='!mb-0'
            >
              {({ field }) => <Input {...field} placeholder='key' />}
            </FieldContainer>
            <FieldContainer
              control={control}
              name={`${subject}.labels.${index}.value`}
              className='flex-1'
              formItemClassName='!mb-0'
            >
              {({ field }) => <Input {...field} placeholder='value' />}
            </FieldContainer>
            <Button
              icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
              onClick={() => labels.remove(index)}
            />
          </Flex>
        ))}
        <Flex align='center' gap={8}>
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faPlus} size='sm' />}
            onClick={() => labels.append({ key: '', value: '' })}
          >
            Add label
          </Button>
          {!labels.fields.length && (
            <Typography.Text type='secondary' className='text-xs'>
              No label constraint.
            </Typography.Text>
          )}
        </Flex>
      </Flex>

      <Typography.Text className='text-sm'>Label expressions</Typography.Text>
      <Flex vertical gap={8} className='mt-2'>
        {expressions.fields.map((expression, index) => {
          const operator = expressionValues?.[index]?.operator ?? '';

          return (
            <Flex key={expression.id} gap={8} align='flex-start'>
              <FieldContainer
                control={control}
                name={`${subject}.matchExpressions.${index}.key`}
                className='flex-1'
                formItemClassName='!mb-0'
              >
                {({ field }) => <Input {...field} placeholder='key' />}
              </FieldContainer>
              <FieldContainer
                control={control}
                name={`${subject}.matchExpressions.${index}.operator`}
                formItemClassName='!mb-0'
              >
                {({ field }) => (
                  <Select {...field} style={{ width: 148 }} options={operatorOptions} />
                )}
              </FieldContainer>
              <FieldContainer
                control={control}
                name={`${subject}.matchExpressions.${index}.values`}
                className='flex-1'
                formItemClassName='!mb-0'
              >
                {({ field }) => (
                  <Select
                    {...field}
                    mode='tags'
                    open={false}
                    suffixIcon={null}
                    className='w-full'
                    disabled={valuelessOperators.includes(operator)}
                    placeholder={valuelessOperators.includes(operator) ? 'No values' : 'values'}
                  />
                )}
              </FieldContainer>
              <Button
                icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                onClick={() => expressions.remove(index)}
              />
            </Flex>
          );
        })}
        <Flex align='center' gap={8}>
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faPlus} size='sm' />}
            onClick={() => expressions.append({ key: '', operator: 'In', values: [] })}
          >
            Add expression
          </Button>
          {!expressions.fields.length && (
            <Typography.Text type='secondary' className='text-xs'>
              No expression constraint.
            </Typography.Text>
          )}
        </Flex>
      </Flex>

      <Button size='small' type='link' className='!px-0 mt-4' onClick={applyToEvery}>
        Clear and apply to every {noun}
      </Button>
    </>
  );
};

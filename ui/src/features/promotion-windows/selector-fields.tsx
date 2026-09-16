import { faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Input, Select, Typography } from 'antd';
import { useController, useFieldArray, useFormContext } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';

import { PromotionWindowFormValues } from './promotion-window-form-utils';

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
  const { control } = useFormContext<PromotionWindowFormValues>();

  const noun = subjectLabel[subject];

  const nameModeField = useController({ control, name: `${subject}.nameMode` }).field;
  const nameMode = nameModeField.value;

  const labels = useFieldArray({ control, name: `${subject}.labels` });

  return (
    <>
      <Typography.Text strong>{noun}s this window applies to</Typography.Text>
      <Typography.Paragraph type='secondary' className='text-xs !mt-1 !mb-4'>
        Leave both empty to match every {noun}. When both are set, a {noun} must match the name and
        every label to be covered by this window.
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
      <Flex vertical gap={8} className='mt-2'>
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
    </>
  );
};

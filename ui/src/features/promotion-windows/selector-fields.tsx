import { faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Input, Segmented, Select, Typography } from 'antd';
import { useController, useFieldArray, useFormContext, useWatch } from 'react-hook-form';

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

  const mode = useWatch({ control, name: `${subject}.mode` });
  const nameModeField = useController({ control, name: `${subject}.nameMode` }).field;
  const nameMode = nameModeField.value;

  const labels = useFieldArray({ control, name: `${subject}.labels` });

  return (
    <>
      <FieldContainer
        control={control}
        name={`${subject}.mode`}
        label={`${noun}s this window applies to`}
      >
        {({ field }) => (
          <Segmented
            {...field}
            options={[
              { label: `Every ${noun}`, value: 'all' },
              { label: 'By name', value: 'name' },
              { label: 'By labels', value: 'labels' }
            ]}
          />
        )}
      </FieldContainer>

      {mode === 'name' && (
        <FieldContainer
          control={control}
          name={`${subject}.name`}
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
      )}

      {mode === 'labels' && (
        <>
          {labels.fields.map((label, index) => (
            <Flex key={label.id} gap={8} align='center'>
              <FieldContainer
                control={control}
                name={`${subject}.labels.${index}.key`}
                className='flex-1'
              >
                {({ field }) => <Input {...field} placeholder='key' />}
              </FieldContainer>
              <FieldContainer
                control={control}
                name={`${subject}.labels.${index}.value`}
                className='flex-1'
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
                An empty selector matches every {noun}.
              </Typography.Text>
            )}
          </Flex>
        </>
      )}
    </>
  );
};

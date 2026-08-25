import { faFlag, faLock, faQuestionCircle, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Button, Card, Flex, Input, Popover, Select, Space, Switch, Typography } from 'antd';

import { ObjectEditor } from '@ui/features/common/object-editor';

import { PolicyDraft, PolicySelectorType, initialPolicy } from '../types';

const selectorTypeOptions: { value: PolicySelectorType; label: string }[] = [
  { value: 'exact', label: 'Exact name' },
  { value: 'regex', label: 'Regex' },
  { value: 'glob', label: 'Glob' },
  { value: 'labels', label: 'Match labels' }
];

type PolicyCardProps = {
  policy: PolicyDraft;
  stageNames: string[];
  onChange: (policy: PolicyDraft) => void;
  onRemove: () => void;
};

const PolicyCard = ({ policy, stageNames, onChange, onRemove }: PolicyCardProps) => {
  const patch = (p: Partial<PolicyDraft>) => onChange({ ...policy, ...p });

  return (
    <div className='rounded-md border border-gray-200 dark:border-neutral-700 p-3'>
      <Flex gap={12} align='flex-start'>
        <div className='w-40 shrink-0'>
          <div className='text-xs font-medium text-gray-600 dark:text-neutral-400 mb-1'>
            Selector
          </div>
          <Select
            className='w-full'
            value={policy.selectorType}
            options={selectorTypeOptions}
            onChange={(selectorType) => patch({ selectorType, value: '' })}
          />
        </div>
        <div className='flex-1 min-w-0'>
          <div className='text-xs font-medium text-gray-600 dark:text-neutral-400 mb-1'>
            {policy.selectorType === 'labels' ? 'Stage labels' : 'Matches'}
          </div>
          {policy.selectorType === 'exact' ? (
            <Select
              className='w-full'
              placeholder='Select a stage'
              value={policy.value || undefined}
              onChange={(value) => patch({ value })}
              options={stageNames.map((name) => ({ value: name, label: name }))}
            />
          ) : policy.selectorType === 'labels' ? (
            <ObjectEditor
              value={policy.matchLabels}
              onChange={(matchLabels) => patch({ matchLabels })}
              keyPlaceholder='env'
              valuePlaceholder='prod'
            />
          ) : (
            <Input
              className='font-mono'
              placeholder={policy.selectorType === 'regex' ? '^prod-.*$' : 'prod-*'}
              value={policy.value}
              onChange={(e) => patch({ value: e.target.value })}
            />
          )}
        </div>
        <div className='shrink-0 text-center'>
          <div className='text-xs font-medium text-gray-600 dark:text-neutral-400 mb-1'>
            Auto-promote
          </div>
          <Switch
            checked={policy.autoPromotionEnabled}
            onChange={(autoPromotionEnabled) => patch({ autoPromotionEnabled })}
          />
        </div>
        <Button
          type='text'
          danger
          size='small'
          className='mt-5'
          icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
          onClick={onRemove}
        />
      </Flex>
    </div>
  );
};

type StepPoliciesProps = {
  value: PolicyDraft[];
  stageNames: string[];
  onChange: (value: PolicyDraft[]) => void;
};

export const StepPolicies = ({ value, stageNames, onChange }: StepPoliciesProps) => {
  if (stageNames.length === 0) {
    return (
      <div className='flex flex-col items-center rounded-lg border border-dashed border-gray-300 dark:border-neutral-600 bg-white dark:bg-neutral-900 py-14 text-center'>
        <FontAwesomeIcon
          icon={faLock}
          className='text-2xl text-gray-300 dark:text-neutral-600 mb-3'
        />
        <div className='text-base font-semibold text-gray-800 dark:text-neutral-100'>
          Add stages first
        </div>
        <div className='text-sm text-gray-500 dark:text-neutral-400 mt-1 max-w-md'>
          Promotion policies act on Stages. Go back to the Stages step to add at least one, then
          return here to configure auto-promotion.
        </div>
      </div>
    );
  }

  const add = () => onChange([...value, initialPolicy(stageNames[0])]);

  return (
    <Card
      type='inner'
      title={
        <Space size={4}>
          Promotion Policies
          <Popover content='A policy decides whether a Stage auto-promotes when new Freight arrives. Without one, every promotion is manual.'>
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      extra={
        <Button icon={<FontAwesomeIcon icon={faFlag} />} onClick={add}>
          Add Policy
        </Button>
      }
    >
      <Alert
        type='warning'
        showIcon
        className='mb-4'
        message='Pattern and label selectors can match Stages added later. Prefer exact names for production unless you have deliberately governed your Stage labels.'
      />
      {value.length === 0 ? (
        <div className='flex flex-col items-center py-8 text-center'>
          <FontAwesomeIcon
            icon={faFlag}
            className='text-2xl text-gray-300 dark:text-neutral-600 mb-3'
          />
          <div className='text-sm text-gray-500 dark:text-neutral-400 max-w-md mb-5'>
            No policies — every promotion stays manual. Most teams auto-promote into their first
            stage and keep production manual.
          </div>
          <Button type='primary' icon={<FontAwesomeIcon icon={faFlag} />} onClick={add}>
            Add Policy
          </Button>
        </div>
      ) : (
        <Flex gap={12} vertical>
          {value.map((policy, i) => (
            <PolicyCard
              key={i}
              policy={policy}
              stageNames={stageNames}
              onChange={(next) => onChange(value.map((p, x) => (x === i ? next : p)))}
              onRemove={() => onChange(value.filter((_, x) => x !== i))}
            />
          ))}
        </Flex>
      )}
    </Card>
  );
};

import { faGear, faObjectGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Popover, Switch } from 'antd';
import { ReactNode, useMemo, useState } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { Stage } from '@ui/gen/api/v2/models';

import { useFreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { groupNodes } from './group-nodes';

type DisplaySetting = {
  key: 'showAlias' | 'showColors';
  label: string;
};

const buttonLabel = 'Display settings';

const settings: DisplaySetting[] = [
  { key: 'showAlias', label: 'Alias' },
  { key: 'showColors', label: 'Colors' }
];

// SettingSwitch is one row of the dropdown: the setting's name, then its
// switch. The label wraps both, so clicking the name flips the switch too.
const SettingSwitch = (props: {
  label: ReactNode;
  checked: boolean;
  onChange(checked: boolean): void;
}) => (
  <label className='flex items-center justify-between gap-6 text-xs cursor-pointer'>
    {props.label}
    <Switch size='small' checked={props.checked} onChange={props.onChange} />
  </label>
);

type DisplaySettingsButtonProps = {
  stages: Stage[];
  warehouses: WarehouseExpanded[];
  pipelineView: 'graph' | 'list';
};

// DisplaySettingsButton holds the preferences that change how the pipeline
// looks rather than which freight or Stages it shows, so they stay apart from
// the filters.
export const DisplaySettingsButton = (props: DisplaySettingsButtonProps) => {
  const ctx = useFreightTimelineControllerContext();
  const [open, setOpen] = useState(false);

  const stackedNodesParents = useMemo(
    () => groupNodes(props.stages, props.warehouses).filter(Boolean),
    [props.stages, props.warehouses]
  );

  if (!ctx) {
    throw new Error('missing context freightTimelineControllerContext');
  }

  const { preferredFilter, setPreferredFilter } = ctx;

  return (
    <Popover
      trigger='click'
      placement='bottomRight'
      open={open}
      onOpenChange={setOpen}
      content={
        <div className='flex flex-col gap-2.5'>
          {settings.map(({ key, label }) => (
            <SettingSwitch
              key={key}
              label={label}
              checked={preferredFilter[key]}
              onChange={(checked) => setPreferredFilter({ ...preferredFilter, [key]: checked })}
            />
          ))}

          <div className='flex items-center justify-between gap-6 text-xs'>
            Group stages
            <Button
              size='small'
              aria-label='Group stages'
              icon={<FontAwesomeIcon icon={faObjectGroup} />}
              onClick={() => {
                setPreferredFilter({ ...preferredFilter, stackedNodesParents });
                // Close so the regrouped graph is not hidden behind the dropdown.
                setOpen(false);
              }}
              disabled={stackedNodesParents.length < 1 || props.pipelineView !== 'graph'}
            />
          </div>
        </div>
      }
    >
      <Button
        aria-label={buttonLabel}
        title={buttonLabel}
        icon={<FontAwesomeIcon icon={faGear} />}
      />
    </Popover>
  );
};

import { faCaretDown, faCaretUp, faCog, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Tag } from 'antd';
import classNames from 'classnames';

import { RunnerWithConfiguration } from './types';

type SelectedRunnerProps = {
  isEditing?: boolean;
  onSettingOpen(): void;
  order: number;
  lastIndexOfOrder: number;
  runner: RunnerWithConfiguration;
  orderMoveUp(): void;
  orderMoveDown(): void;
  onDelete(): void;
};

export const SelectedRunner = (props: SelectedRunnerProps) => {
  return (
    <Card
      className={classNames('cursor-pointer', {
        'shadow-sm': props.isEditing,
        'border-gray-400': props.isEditing
      })}
      size='small'
      onClick={props.onSettingOpen}
    >
      <div className='flex items-center gap-5'>
        <div className='flex flex-col gap-5 text-2xl cursor-pointer'>
          {props.order > 0 && (
            <FontAwesomeIcon
              icon={faCaretUp}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                props.orderMoveUp();
              }}
            />
          )}
          {props.order < props.lastIndexOfOrder && (
            <FontAwesomeIcon
              icon={faCaretDown}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                props.orderMoveDown();
              }}
            />
          )}
        </div>

        <span className='font-semibold'>
          {props.order + 1} - {props.runner.identifier}
        </span>

        {!!props.runner?.as && (
          <Tag className='text-xs max-w-20 overflow-hidden ml-auto' color='blue'>
            {props.runner.as}
          </Tag>
        )}

        <div className={classNames('space-x-4', { 'ml-auto': !props.runner?.as })}>
          <Button icon={<FontAwesomeIcon icon={faCog} />} onClick={props.onSettingOpen} />
          <Button
            icon={<FontAwesomeIcon icon={faTrash} className='text-red-500' />}
            onClick={(e) => {
              e.stopPropagation();
              props.onDelete();
            }}
          />
        </div>
      </div>
    </Card>
  );
};

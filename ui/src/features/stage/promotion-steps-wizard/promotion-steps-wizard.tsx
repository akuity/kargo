import {
  faAdd,
  faCaretDown,
  faCaretUp,
  faCog,
  faEye,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tag } from 'antd';
import Card from 'antd/es/card/Card';
import classNames from 'classnames';
import { useState } from 'react';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { useModal } from '@ui/features/common/modal/use-modal';
import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';

import { PromotionStepDrawer } from './promotion-step-drawer';
import { RunnerWithConfiguration } from './types';
import { isRunnersEqual } from './utils';

export type PromotionStepsWizardType = {
  // let the YAML -> state transformation dance delegate to component up in the tree
  steps: RunnerWithConfiguration[];
  onChange: (newStepsState: RunnerWithConfiguration[]) => void;
};

export const PromotionStepsWizard = (props: PromotionStepsWizardType) => {
  const { registry } = usePromotionDirectivesRegistryContext();

  const selectedRunners = props.steps;
  const setSelectedRunners = props.onChange;

  const runnerEdit = useModal();

  const [viewingRunner, setViewingRunner] = useState<RunnerWithConfiguration | undefined>();

  const liveViewingRunner: RunnerWithConfiguration | undefined =
    viewingRunner && selectedRunners?.find((r) => isRunnersEqual(r, viewingRunner));

  return (
    <div>
      {/* AVAILABLE RUNNERS - TODO(Marvin9) add search functionality */}
      <div className='flex gap-2 flex-wrap'>
        {registry.runners
          // .filter((runner) => !selectedRunners.find((r) => r.identifier === runner.identifier))
          // .slice(0, 5)
          .map((runner, idx) => (
            <Tag
              className='cursor-pointer'
              key={runner.identifier + idx}
              onClick={() => {
                setSelectedRunners([...selectedRunners, runner]);
              }}
              icon={<FontAwesomeIcon className='mr-2' icon={faAdd} />}
            >
              {runner.identifier}

              {runner.unstable_icons.length > 0 &&
                runner.unstable_icons.map((icon) => (
                  <FontAwesomeIcon className='ml-2' key={icon.iconName} icon={icon} />
                ))}
            </Tag>
          ))}
      </div>

      {/* SELECTED RUNNERS */}
      {selectedRunners.length > 0 && (
        <div className='flex gap-5 relative'>
          <div className='mt-5 space-y-4 w-6/12'>
            {selectedRunners.map((runner, order) => {
              const patchRunner = (nextRunner: RunnerWithConfiguration) => {
                setSelectedRunners(
                  selectedRunners.map((localRunner, localOrder) => {
                    if (order === localOrder) {
                      return {
                        ...localRunner,
                        ...nextRunner
                      };
                    }

                    return localRunner;
                  })
                );
              };

              const onSettingOpen = () => {
                runnerEdit.show((props) => (
                  <PromotionStepDrawer
                    {...props}
                    selectedRunner={runner}
                    patchSelectedRunner={patchRunner}
                  />
                ));
              };

              return (
                <Card
                  key={runner.identifier}
                  className={classNames('cursor-pointer', {
                    'border-gray-500': isRunnersEqual(runner, viewingRunner)
                  })}
                  size='small'
                  onClick={onSettingOpen}
                >
                  <div className='flex items-center gap-5'>
                    <div className='flex flex-col gap-5 text-2xl cursor-pointer'>
                      {order > 0 && (
                        <FontAwesomeIcon
                          icon={faCaretUp}
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            const newOrder = [...selectedRunners];
                            newOrder[order] = selectedRunners[order - 1];
                            newOrder[order - 1] = runner;
                            setSelectedRunners(newOrder);
                          }}
                        />
                      )}
                      {order < selectedRunners.length - 1 && (
                        <FontAwesomeIcon
                          icon={faCaretDown}
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            const newOrder = [...selectedRunners];
                            newOrder[order] = selectedRunners[order + 1];
                            newOrder[order + 1] = runner;
                            setSelectedRunners(newOrder);
                          }}
                        />
                      )}
                    </div>

                    <span className='font-semibold'>
                      {order + 1} - {runner.identifier}
                    </span>

                    {runner.unstable_icons.length > 0 && (
                      <div className='flex ml-auto gap-2'>
                        {runner.unstable_icons.map((icon) => (
                          <FontAwesomeIcon key={icon.iconName} icon={icon} />
                        ))}
                      </div>
                    )}

                    <div className='space-x-4'>
                      <Button icon={<FontAwesomeIcon icon={faCog} />} onClick={onSettingOpen} />
                      <Button
                        icon={<FontAwesomeIcon icon={faTrash} className='text-red-500' />}
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedRunners(
                            selectedRunners.filter((r) => r.identifier !== runner.identifier)
                          );
                        }}
                      />
                      <Button
                        icon={<FontAwesomeIcon icon={faEye} />}
                        onClick={(e) => {
                          e.stopPropagation();
                          setViewingRunner(runner);
                        }}
                      />
                    </div>
                  </div>
                </Card>
              );
            })}
          </div>

          {viewingRunner && liveViewingRunner && (
            <Card
              color='lightgray'
              className='w-6/12 mt-5 h-fit sticky top-0'
              size='small'
              title={
                <>
                  <FontAwesomeIcon icon={faEye} className='mr-2' />
                  {liveViewingRunner.identifier}{' '}
                  {liveViewingRunner?.as && (
                    <Tag color='blue' className='ml-4'>
                      {liveViewingRunner.as}
                    </Tag>
                  )}
                </>
              }
            >
              <YamlEditor
                value={JSON.stringify(liveViewingRunner?.state, null, ' ')}
                height='350px'
                disabled
                key={JSON.stringify(liveViewingRunner?.state)}
              />
            </Card>
          )}
        </div>
      )}
    </div>
  );
};

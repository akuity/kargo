import {
  faAdd,
  faCaretDown,
  faCaretUp,
  faPencil,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tag } from 'antd';
import Card from 'antd/es/card/Card';

import { useModal } from '@ui/features/common/modal/use-modal';
import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';

import { PromotionStepDrawer } from './promotion-step-drawer';
import { RunnerWithConfiguration } from './types';

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

  return (
    <div>
      {/* AVAILABLE RUNNERS - TODO(Marvin9) add search functionality */}
      <div className='flex gap-2 flex-wrap'>
        {registry.runners
          .filter((runner) => !selectedRunners.find((r) => r.identifier === runner.identifier))
          // .slice(0, 5)
          .map((runner, idx) => (
            <Tag
              className='cursor-pointer'
              color={'cyan'}
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
        <div className='mt-5 space-y-4'>
          {selectedRunners.map((runner, order) => (
            <Card key={runner.identifier}>
              <div className='flex items-center gap-5 w-full'>
                <div className='flex flex-col gap-5 text-2xl cursor-pointer'>
                  {order > 0 && (
                    <FontAwesomeIcon
                      icon={faCaretUp}
                      onClick={() => {
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
                      onClick={() => {
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
                  <Button
                    icon={<FontAwesomeIcon icon={faPencil} />}
                    onClick={() => {
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

                      runnerEdit.show((props) => (
                        <PromotionStepDrawer
                          {...props}
                          selectedRunner={runner}
                          patchSelectedRunner={patchRunner}
                        />
                      ));
                    }}
                  />
                  <Button
                    icon={<FontAwesomeIcon icon={faTrash} className='text-red-500' />}
                    onClick={() =>
                      setSelectedRunners(
                        selectedRunners.filter((r) => r.identifier !== runner.identifier)
                      )
                    }
                  />
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

import { Checkbox, Form, Input } from 'antd';

import { RunnerForm } from './runner-form';
import { RunnerWithConfiguration } from './types';
import { STEP_DESCRIPTION_MAX_LENGTH } from './utils';

export const PromotionStepForm = (props: {
  selectedRunner: RunnerWithConfiguration;
  patchSelectedRunner(next: RunnerWithConfiguration): void;
}) => {
  return (
    <>
      <Form layout='vertical' onSubmitCapture={(e) => e.preventDefault()}>
        <Form.Item label='As'>
          <Input
            value={props.selectedRunner?.as || ''}
            onChange={(e) =>
              props.patchSelectedRunner({ ...props.selectedRunner, as: e.target.value })
            }
          />
        </Form.Item>
        <Form.Item label='Description' extra='Optional. Explains the purpose of this step.'>
          <Input
            value={props.selectedRunner?.description || ''}
            maxLength={STEP_DESCRIPTION_MAX_LENGTH}
            showCount
            onChange={(e) =>
              props.patchSelectedRunner({ ...props.selectedRunner, description: e.target.value })
            }
          />
        </Form.Item>
        <Form.Item extra='When enabled, any failure in this step will not immediately halt execution of the promotion process, nor will it prevent the promotion process, overall, from succeeding.'>
          <Checkbox
            checked={props.selectedRunner?.continueOnError || false}
            onChange={(e) =>
              props.patchSelectedRunner({
                ...props.selectedRunner,
                continueOnError: e.target.checked
              })
            }
          >
            Continue On Error
          </Checkbox>
        </Form.Item>
      </Form>
      <RunnerForm
        runner={props.selectedRunner}
        onSubmit={(newState) => {
          props.patchSelectedRunner({ ...props.selectedRunner, state: newState });
        }}
      />
    </>
  );
};

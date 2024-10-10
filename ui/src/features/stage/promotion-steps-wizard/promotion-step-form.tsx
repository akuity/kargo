import { Form, Input } from 'antd';

import { RunnerForm } from './runner-form';
import { RunnerWithConfiguration } from './types';

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

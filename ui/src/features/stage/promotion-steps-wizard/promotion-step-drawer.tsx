import { Drawer, Form, Input } from 'antd';
import { useState } from 'react';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';

import { RunnerForm } from './runner-form';
import { RunnerWithConfiguration } from './types';

export const PromotionStepDrawer = (
  props: ModalComponentProps & {
    selectedRunner: RunnerWithConfiguration;
    patchSelectedRunner(next: RunnerWithConfiguration): void;
  }
) => {
  const [as, setAs] = useState(props.selectedRunner?.as);

  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      title={props.selectedRunner.identifier}
      width='1400px'
    >
      <Form layout='vertical' onSubmitCapture={(e) => e.preventDefault()}>
        <Form.Item label='As'>
          <Input value={as} onChange={(e) => setAs(e.target.value)} />
        </Form.Item>
      </Form>
      <RunnerForm
        runner={props.selectedRunner}
        onSubmit={(newState) => {
          props.patchSelectedRunner({ ...props.selectedRunner, as, state: newState });
          props.hide();
        }}
      />
    </Drawer>
  );
};

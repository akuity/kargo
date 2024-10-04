import Form, { Templates } from '@rjsf/antd';
import validator from '@rjsf/validator-ajv8';
import { Button, Collapse } from 'antd';
import Alert from 'antd/es/alert/Alert';

import { ErrorBoundary } from '@ui/features/common/layout/error-boundary';

import styles from './runner-form.module.less';
import { RunnerWithConfiguration } from './types';

export type RunnerFormType = {
  runner: RunnerWithConfiguration;
  onSubmit(
    runnerConfig: object /* this is dynamic config that we should not care about and pass to YAML as it is */
  ): void;
};

export const RunnerForm = (props: RunnerFormType) => {
  return (
    <ErrorBoundary
      errorRender={
        <Alert
          message={
            <>
              It looks like there was an error with JSON schema for runner{' '}
              <b>{props.runner.identifier}</b>
            </>
          }
          type='error'
        />
      }
    >
      <div className={styles.container}>
        <Form
          schema={props.runner.config}
          validator={validator}
          onSubmit={(d) => {
            props.onSubmit(d.formData);
          }}
          ref={(attach) => {
            // save yourself from infinite re-renders by not settings this state using 'formData' props
            if (props.runner.state) {
              attach?.setState({ ...attach.state, formData: props.runner.state });
            }
          }}
          uiSchema={{
            'ui:submitButtonOptions': {
              norender: true
            }
          }}
          templates={{
            ObjectFieldTemplate: (props) => {
              if (!Templates.ObjectFieldTemplate) {
                throw new Error('[BUG]: Templates.ObjectFieldTemplate is undefined');
              }

              if (props.title) {
                return (
                  <Collapse
                    items={[
                      {
                        children: <Templates.ObjectFieldTemplate {...props} />,
                        label: props.title
                      }
                    ]}
                  />
                );
              }

              return <Templates.ObjectFieldTemplate {...props} />;
            }
          }}
        >
          <div className='absolute bottom-0 h-10 w-full pt-1 bg-white'>
            <Button htmlType='submit' type='primary'>
              Update
            </Button>
          </div>
        </Form>
      </div>
    </ErrorBoundary>
  );
};

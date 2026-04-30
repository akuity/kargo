import Form from '@rjsf/antd';
import validator from '@rjsf/validator-ajv8';
import Alert from 'antd/es/alert/Alert';

import { ObjectFieldTemplate } from '@ui/features/common/form/rjsf/object-field-template';
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
          onChange={(d) => {
            props.onSubmit(d.formData);
          }}
          formData={props.runner.state}
          uiSchema={{
            'ui:submitButtonOptions': {
              norender: true
            }
          }}
          templates={{
            ObjectFieldTemplate
          }}
        />
      </div>
    </ErrorBoundary>
  );
};

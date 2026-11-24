import Form, { FormProps } from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';
import { JSONSchema7 } from 'json-schema';
import { describe, expect, test } from 'vitest';

import argocdUpdateConfig from '../../../gen/directives/argocd-update-config.json';
import gitCloneConfig from '../../../gen/directives/git-clone-config.json';
import helmTemplateConfig from '../../../gen/directives/helm-template-config.json';

import { RunnerWithConfiguration } from './types';
import { runnerFormData, stepDescriptionForManifest } from './utils';

test('stepDescriptionForManifest', () => {
  expect(stepDescriptionForManifest('does a thing')).toBe('does a thing');
  expect(stepDescriptionForManifest('  does a thing \n')).toBe('does a thing');

  // blank values are omitted: the API rejects an empty description
  expect(stepDescriptionForManifest(undefined)).toBeUndefined();
  expect(stepDescriptionForManifest('')).toBeUndefined();
  expect(stepDescriptionForManifest('   ')).toBeUndefined();
});

describe('runnerFormData', () => {
  const gitClone: RunnerWithConfiguration = {
    identifier: 'git-clone',
    config: gitCloneConfig as JSONSchema7,
    state: {
      repoURL: 'https://github.com/foo/baz',
      checkout: [{ branch: 'main', path: './repo' }]
    }
  };

  // freshly added from the registry: no config entered yet
  const helmTemplate: RunnerWithConfiguration = {
    identifier: 'helm-template',
    config: helmTemplateConfig as JSONSchema7
  };
  const argocdUpdate: RunnerWithConfiguration = {
    identifier: 'argocd-update',
    config: argocdUpdateConfig as JSONSchema7
  };

  test('passes existing config through', () => {
    expect(runnerFormData(gitClone)).toBe(gitClone.state);
  });

  test('a newly added step does not inherit config from the previous step', () => {
    let reported: Record<string, unknown> | undefined;
    const formProps = (runner: RunnerWithConfiguration): FormProps => ({
      schema: runner.config,
      validator,
      formData: runnerFormData(runner),
      onChange: (e) => {
        reported = e.formData;
      }
    });

    // The steps editor reuses one RJSF form for every step, so opening a step
    // re-renders that form with the step's props. Replay React's update
    // lifecycle and return the config the step ends up with: what RJSF reports
    // as the step opens, otherwise what it would report on the first edit.
    const form = new Form(formProps(gitClone));
    form.setState = () => {};
    const openStep = (runner: RunnerWithConfiguration) => {
      reported = undefined;
      const prevProps = form.props;
      const prevState = form.state;
      (form as { props: FormProps }).props = formProps(runner);
      const snapshot = form.getSnapshotBeforeUpdate(prevProps, prevState);
      form.componentDidUpdate(prevProps, prevState, snapshot);
      if (snapshot.shouldUpdate) {
        form.state = snapshot.nextState;
      }
      return reported ?? form.state.formData;
    };

    // fields in a step's config that its own schema does not define
    const foreignFields = (runner: RunnerWithConfiguration, config: object) =>
      Object.keys(config).filter((field) => !(field in (runner.config.properties ?? {})));

    // helm-template has defaults, so its config is reported as soon as it opens
    const helmTemplateConfigured = { ...helmTemplate, state: openStep(helmTemplate) };
    expect(foreignFields(helmTemplate, helmTemplateConfigured.state)).toEqual([]);

    // the editor stores the reported config and re-renders with it
    openStep(helmTemplateConfigured);

    // argocd-update has no defaults: a leak would surface on its first edit
    expect(openStep(argocdUpdate)).toEqual({});
  });
});

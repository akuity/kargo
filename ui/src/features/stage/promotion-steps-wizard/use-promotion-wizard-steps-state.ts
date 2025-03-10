import { useState } from 'react';
import { parse, stringify } from 'yaml';

import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';
import { Runner } from '@ui/features/promotion-directives/registry/types';
import { PromotionStep } from '@ui/gen/api/v1alpha1/generated_pb';
import { PlainMessage } from '@ui/utils/connectrpc-utils';

import { RunnerWithConfiguration } from './types';

// separation is required as each component has separate requirement for YAML <-> Wizard transformation
export const usePromotionWizardStepsState = (initialValue: string | PromotionStep[] = '') => {
  // TODO: move this to props instead?
  const { registry } = usePromotionDirectivesRegistryContext();

  const [runnersState, setRunnersState] = useState(() => {
    if (typeof initialValue === 'string') {
      return yamlToState(initialValue || '', registry.runners);
    }

    return APIPromotionStepsToLocalStateEquivalent(initialValue || [], registry.runners);
  });

  return {
    state: runnersState,
    onChange: setRunnersState,
    getYAML: () => stateToYAML(runnersState),
    setYAML: (yaml: string) => setRunnersState(yamlToState(yaml, registry.runners))
  };
};

const APIPromotionStepsToLocalStateEquivalent = (
  steps: PromotionStep[],
  runnersRegistry: Runner[]
): RunnerWithConfiguration[] => {
  const runnerWithConfig: RunnerWithConfiguration[] = [];

  for (const step of steps) {
    const runnerMeta = runnersRegistry.find((r) => step.uses === r.identifier);

    if (!runnerMeta) {
      // there is no point of having wizard when the necessary values are missing
      throw new Error(
        `could not discover runner ${step.uses}, please use YAML to define promotion steps`
      );
    }

    runnerWithConfig.push({
      ...runnerMeta,
      state: step.config,
      as: step.as
    });
  }

  return runnerWithConfig;
};

/**
 *
 * @param stepsYaml
 * - uses: git-clone
 *   as: foo-bar
 *   config:
 *      ....object
 * @param runnersRegistry
 */
const yamlToState = (stepsYaml: string, runnersRegistry: Runner[]): RunnerWithConfiguration[] => {
  const steps: PromotionStep[] = parse(stepsYaml) || [];

  return APIPromotionStepsToLocalStateEquivalent(steps, runnersRegistry);
};

const stateToYAML = (state: RunnerWithConfiguration[]): string => {
  const promotionSteps: PlainMessage<PromotionStep>[] = [];

  for (const step of state) {
    promotionSteps.push({
      uses: step.identifier,
      // @ts-expect-error this will be object but its hard to convey in types after migration of connectrpc to v2
      config: step.state,
      as: step.as || '',
      vars: []
    });
  }

  return stringify(promotionSteps);
};

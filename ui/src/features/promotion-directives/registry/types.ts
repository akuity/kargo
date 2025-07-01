// central registry with available/installed runners

import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { JSONSchema7 } from 'json-schema';

// runners are basically what you can configure in promotionTemplates.spec.stages in Stage resource
export type PromotionDirectivesRegistry = {
  runners: Runner[];
};

// runner is source of truth for all configuration and metadata related to installed runner
export type Runner = {
  // unique identifier such that kargo knows which runner to operate
  // example - git-clone, git-clear
  identifier: string;
  // UI helper
  // this accepts font-awesome icon
  unstable_icons?: IconDefinition[];
  config: JSONSchema7;
};

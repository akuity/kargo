import { Freight } from '@ui/gen/v1alpha1/generated_pb';

export const ALIAS_LABEL_KEY = 'kargo.akuity.io/alias';

export const getAlias = (freight?: Freight): string | undefined => {
  return freight?.metadata?.labels[ALIAS_LABEL_KEY] || undefined;
};

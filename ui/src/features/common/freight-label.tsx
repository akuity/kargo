import { Freight } from '@ui/gen/v1alpha1/types_pb';

import { ALIAS_LABEL_KEY } from './config';

export const FreightLabel = ({ freight }: { freight?: Freight }) => (
  <>
    {freight?.metadata?.labels[ALIAS_LABEL_KEY]
      ? freight?.metadata?.labels[ALIAS_LABEL_KEY]
      : freight?.metadata?.name?.substring(0, 7) || 'N/A'}
  </>
);

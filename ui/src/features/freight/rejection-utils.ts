import type { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import type { PlainMessageRecursive } from '@ui/utils/connectrpc-utils';

export const isFreightRejected = (freight?: Freight | PlainMessageRecursive<Freight>) =>
  !!freight?.status?.rejected;

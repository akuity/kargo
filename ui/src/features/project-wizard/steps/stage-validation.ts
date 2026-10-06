import { dnsRegex } from '@ui/features/common/utils';

import { StageDraft } from '../types';

// A Stage needs a valid name and at least one requested Freight, mirroring the
// app's Create Stage form ("Requested Freight are required to create a Stage").
export const isValidStage = (s: StageDraft) =>
  dnsRegex.test(s.name) && s.requestedFreight.length > 0;

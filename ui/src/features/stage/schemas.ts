import { z } from 'zod';

import { zodValidators } from '@ui/utils/validators';

export const requestedFreightSchema = z.object({
  warehouse: zodValidators.requiredString,
  sources: z.object({
    direct: z.boolean(),
    upstreamStages: z.array(z.string())
  })
});

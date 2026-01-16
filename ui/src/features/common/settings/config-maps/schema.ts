import z from 'zod';

import { zodValidators } from '@ui/utils/validators';

export const confgMapSchema = z.object({
  name: zodValidators.requiredString,
  data: z.record(z.string(), zodValidators.requiredString),
  description: z.string().optional()
});

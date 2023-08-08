import { z } from 'zod';

export const validatorMessages = {
  required: 'This field is required'
};

export const zodValidators = {
  requiredString: z
    .string({ required_error: validatorMessages.required })
    .min(1, { message: validatorMessages.required })
};

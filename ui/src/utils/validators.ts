import { z } from 'zod';

export const validatorMessages = {
  required: 'This field is required'
};

export const zodValidators = {
  requiredString: z
    .string({
      error: (issue) => (issue.input === undefined ? validatorMessages.required : issue.message)
    })
    .min(1, { error: validatorMessages.required })
};

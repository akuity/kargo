import { z } from 'zod';

const stepOutputLinkSchema = z.object({
  url: z.string().min(1),
  icon: z.string().optional(),
  tooltip: z.string().optional(),
  label: z.string().optional()
});

export type StepOutputLink = z.infer<typeof stepOutputLinkSchema>;

export function getOutputLinks(output: Record<string, unknown>): StepOutputLink[] {
  const links = (output as { links?: unknown })?.links;
  if (!Array.isArray(links) || links.length === 0) return [];
  return links.flatMap((l) => {
    const result = stepOutputLinkSchema.safeParse(l);
    return result.success ? [result.data] : [];
  });
}

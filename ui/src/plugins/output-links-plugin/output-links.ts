export interface StepOutputLink {
  url: string;
  icon?: string;
  tooltip?: string;
  label?: string;
}

export function getOutputLinks(output: Record<string, unknown>): StepOutputLink[] {
  const links = (output as { links?: unknown })?.links;
  if (!Array.isArray(links) || links.length === 0) return [];
  return links.filter(
    (l): l is StepOutputLink =>
      l !== null && typeof l === 'object' && typeof l.url === 'string' && l.url !== ''
  );
}

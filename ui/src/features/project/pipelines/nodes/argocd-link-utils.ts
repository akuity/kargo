import z from 'zod';

const argoCDContextSchema = z.array(
  z.object({
    name: z.string(),
    namespace: z.string()
  })
);

export type ArgoCDContext = z.infer<typeof argoCDContextSchema>[number];

export const argoCDAppKey = (app: ArgoCDContext) => `${app.namespace}/${app.name}`;

export const parseArgoCDContext = (rawValue?: string): ArgoCDContext[] => {
  if (!rawValue) {
    return [];
  }

  const apps = argoCDContextSchema.parse(JSON.parse(rawValue));
  const seen = new Set<string>();

  return apps.filter((app) => {
    const key = argoCDAppKey(app);
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
};

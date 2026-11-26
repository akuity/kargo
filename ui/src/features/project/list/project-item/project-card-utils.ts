import type { Project } from '@ui/gen/api/v2/models';

// Relative: vitest resolves @ui paths to files but not to a directory's index.
import { WarehouseColorMapHex } from '../../../stage/utils';

// The Warehouse palette is the one tuned for the tinted tile a project card
// shares with the Warehouse node: every color in it clears 4.5:1 against its
// own tile, in either theme.
const accentPalette = Object.values(WarehouseColorMapHex);

// projectAccent picks a project's color from its name. It is identity, not
// status -- the same project is the same color on every visit and for every
// reader -- so it is derived rather than stored.
export const projectAccent = (name: string): string => {
  let hash = 0;
  for (const char of name) {
    hash = (hash * 31 + (char.codePointAt(0) || 0)) | 0;
  }
  return accentPalette[Math.abs(hash) % accentPalette.length];
};

export type ProjectReadiness =
  | { state: 'ready' }
  | { state: 'syncing'; message?: string }
  | { state: 'failed'; message?: string };

// projectReadiness reads whether the controller has finished setting the
// project up. A project being set up says Syncing until it is Ready; one that
// cannot be set up says why, either as a Ready condition with some other
// reason or as a Stalled one. A project with no conditions at all is not
// treated as broken: there is nothing to say it is.
export const projectReadiness = (project?: Project): ProjectReadiness => {
  const conditions = project?.status?.conditions || [];

  const stalled = conditions.find((c) => c.type === 'Stalled' && c.status === 'True');
  if (stalled) {
    return { state: 'failed', message: stalled.message };
  }

  const ready = conditions.find((c) => c.type === 'Ready');
  if (!ready || ready.status === 'True') {
    return { state: 'ready' };
  }

  if (ready.status === 'Unknown' || ready.reason === 'Syncing') {
    return { state: 'syncing', message: ready.message };
  }

  return { state: 'failed', message: ready.message };
};

export type HealthSummary = {
  count: number;
  healthy: number;
  allHealthy: boolean;
};

// healthSummary condenses one of a project's stats blocks. The controller
// counts only what is explicitly healthy, so the remainder is not unhealthy --
// a Stage nothing has been promoted to yet has no health at all -- and the
// summary says how many are healthy rather than how many are failing.
export const healthSummary = (stats?: {
  count?: number;
  health?: { healthy?: number };
}): HealthSummary => {
  const count = Math.max(stats?.count || 0, 0);
  const healthy = Math.min(Math.max(stats?.health?.healthy || 0, 0), count);

  return {
    count,
    healthy,
    allHealthy: count > 0 && healthy === count
  };
};

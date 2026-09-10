// The Stage drawer's tabs are addressable by URL: /project/:name/stage/:stageName/:tab.
// Keys double as the URL segment, so they are lowercase slugs rather than the
// labels shown on screen.
export const StageTab = {
  PROMOTIONS: 'promotions',
  VERIFICATIONS: 'verifications',
  LIVE_MANIFEST: 'live-manifest',
  FREIGHT_HISTORY: 'freight-history',
  SETTINGS: 'settings'
} as const;

export type StageTabKey = (typeof StageTab)[keyof typeof StageTab];

export const DEFAULT_STAGE_TAB: StageTabKey = StageTab.PROMOTIONS;

const builtInTabs = new Set<string>(Object.values(StageTab));

// extensionTabKey derives a URL-safe key for a tab an extension contributes,
// from its label. Keys already taken (by a built-in tab or an earlier
// extension) get a numeric suffix so two extensions with the same label stay
// distinct.
export const extensionTabKey = (label: string, taken: Set<string>): string => {
  const base =
    label
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '') || 'tab';
  let key = base;
  for (let n = 2; taken.has(key); n++) {
    key = `${base}-${n}`;
  }
  return key;
};

// resolveStageTab maps the :tab route segment to the tab to show. An absent or
// unrecognized segment falls back to the default rather than an empty pane, so
// stale links still land somewhere useful.
export const resolveStageTab = (
  segment: string | undefined,
  extensionKeys: Iterable<string> = []
): string => {
  if (!segment) {
    return DEFAULT_STAGE_TAB;
  }
  if (builtInTabs.has(segment)) {
    return segment;
  }
  for (const key of extensionKeys) {
    if (key === segment) {
      return segment;
    }
  }
  return DEFAULT_STAGE_TAB;
};

import { PolicyDraft } from '../types';

// A policy needs a target: a non-empty name/pattern, or at least one label.
export const isValidPolicy = (p: PolicyDraft) =>
  p.selectorType === 'labels' ? Object.keys(p.matchLabels).length > 0 : p.value.trim().length > 0;

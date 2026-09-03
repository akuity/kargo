import { faCircleExclamation, faHourglassStart } from '@fortawesome/free-solid-svg-icons';
import { describe, expect, test } from 'vitest';

import {
  getPromotionPhasePresentation,
  isPromotionPhase,
  promotionPhasePresentations,
  promotionPhases
} from './promotion-phase';

describe('isPromotionPhase()', () => {
  test.each([
    ['Succeeded', true],
    ['Aborted', true],
    ['succeeded', false],
    ['Unknown', false],
    ['', false],
    [undefined, false]
  ])('%s -> %s', (phase, expected) => {
    expect(isPromotionPhase(phase)).toBe(expected);
  });
});

describe('getPromotionPhasePresentation()', () => {
  test('returns the presentation for a known phase', () => {
    expect(getPromotionPhasePresentation('Running')).toBe(promotionPhasePresentations.Running);
  });

  test('Failed and Errored look alike', () => {
    const errored = getPromotionPhasePresentation('Errored');
    expect(errored.icon).toBe(faCircleExclamation);
    expect(errored.tagColor).toBe('error');
    expect(getPromotionPhasePresentation('Failed')).toEqual(errored);
  });

  test.each([undefined, '', 'Unknown'])('%s falls back to Pending', (phase) => {
    const presentation = getPromotionPhasePresentation(phase);
    expect(presentation).toBe(promotionPhasePresentations.Pending);
    expect(presentation.icon).toBe(faHourglassStart);
    expect(presentation.spin).toBe(false);
  });

  test('only Running spins', () => {
    const spinning = promotionPhases.filter((phase) => getPromotionPhasePresentation(phase).spin);
    expect(spinning).toEqual(['Running']);
  });
});

describe('promotionPhases', () => {
  test('covers every phase exactly once', () => {
    expect([...promotionPhases].sort()).toEqual(Object.keys(promotionPhasePresentations).sort());
  });
});

import { faCircleExclamation, faHourglassStart } from '@fortawesome/free-solid-svg-icons';
import { describe, expect, test } from 'vitest';

import {
  getPromotionPhasePresentation,
  getRoundTargetPhasePresentation,
  isPromotionPhase,
  notAttempted,
  promotionPhasePresentations,
  promotionPhases,
  roundTargetPhaseLabel,
  roundTargetPhases,
  unreachedPhase
} from './promotion-phase';

describe('isPromotionPhase()', () => {
  test.each([
    ['Succeeded', true],
    ['Aborted', true],
    ['succeeded', false],
    ['Unknown', false],
    // Names every object inherits are not phases.
    ['toString', false],
    ['constructor', false],
    ['__proto__', false],
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

  test.each([undefined, '', 'Unknown', 'toString'])('%s falls back to Pending', (phase) => {
    const presentation = getPromotionPhasePresentation(phase);
    expect(presentation).toBe(promotionPhasePresentations.Pending);
    expect(presentation.icon).toBe(faHourglassStart);
    expect(presentation.spin).toBe(false);
  });

  test('only Running spins', () => {
    const spinning = promotionPhases.filter((phase) => getPromotionPhasePresentation(phase).spin);
    expect(spinning).toEqual(['Running']);
  });

  test('each phase names the theme token it is painted with', () => {
    const tokens = Object.fromEntries(
      promotionPhases.map((phase) => [phase, getPromotionPhasePresentation(phase).colorToken])
    );
    expect(tokens).toEqual({
      Succeeded: 'colorSuccess',
      Running: 'colorInfo',
      Pending: 'colorFillSecondary',
      Failed: 'colorError',
      Errored: 'colorError',
      Aborted: 'colorTextQuaternary'
    });
  });
});

describe('promotionPhases', () => {
  test('covers every phase exactly once', () => {
    expect([...promotionPhases].sort()).toEqual(Object.keys(promotionPhasePresentations).sort());
  });
});

describe('unreachedPhase()', () => {
  test.each([
    // A round still under way has yet to reach the Target.
    [undefined, 'Pending'],
    ['', 'Pending'],
    ['Pending', 'Pending'],
    ['Running', 'Pending'],
    // A round that ended without succeeding never will: it exceeded the
    // failures it tolerates, or was aborted, and stopped fanning out.
    ['Failed', 'NotAttempted'],
    ['Errored', 'NotAttempted'],
    ['Aborted', 'NotAttempted'],
    // A round that succeeded promoted to every Target it names.
    ['Succeeded', 'Succeeded'],
    // A phase this UI does not know is not taken for an ending.
    ['Rethinking', 'Pending']
  ])('a Target with no Promotion in a round that is %s is %s', (roundPhase, expected) => {
    expect(unreachedPhase(roundPhase)).toBe(expected);
  });
});

describe('what a round reports for a Target', () => {
  test('is a Promotion phase, or that the Target was not attempted', () => {
    expect(roundTargetPhases).toEqual([...promotionPhases, notAttempted]);
    // Not a phase a Promotion can be in, so nothing that reads a Promotion
    // ever shows it.
    expect(isPromotionPhase(notAttempted)).toBe(false);
  });

  test('not attempted is neither an outcome nor something still to come', () => {
    const presentation = getRoundTargetPhasePresentation(notAttempted);
    expect(presentation.tagColor).toBe('default');
    expect(presentation.spin).toBe(false);
    expect(presentation.colorToken).toBe('colorTextQuaternary');
    // In particular it does not fall back to Pending, as an unknown phase does.
    expect(presentation).not.toBe(promotionPhasePresentations.Pending);
    expect(presentation.colorToken).not.toBe(promotionPhasePresentations.Pending.colorToken);
  });

  test.each([...promotionPhases, undefined, 'Unknown'])(
    '%s is presented as a Promotion is',
    (p) => {
      expect(getRoundTargetPhasePresentation(p)).toBe(getPromotionPhasePresentation(p));
    }
  );

  test('is named by its phase, and in words when it was not attempted', () => {
    expect(roundTargetPhaseLabel('Failed')).toBe('Failed');
    expect(roundTargetPhaseLabel(notAttempted)).toBe('Not attempted');
  });
});

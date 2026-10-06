import { expect, test } from 'vitest';

import {
  WizardState,
  initialCredential,
  initialPolicy,
  initialWizardState,
  initialWarehouse
} from '../types';

import { firstIncompleteStep, stepDataValidity } from './step-validity';

// A draft that is ready to apply: a valid project name and nothing else, since
// every other step is optional.
const readyState = (overrides: Partial<WizardState> = {}): WizardState => ({
  ...initialWizardState(),
  basics: { name: 'my-project', description: '' },
  ...overrides
});

const validCredential = () => ({
  ...initialCredential('git'),
  name: 'github-creds',
  repoURL: 'https://github.com/akuity/kargo.git',
  username: 'someone',
  password: 'secret'
});

test('stepDataValidity reports every step, and skipped optional steps are valid', () => {
  const validity = stepDataValidity(readyState());
  // The non-partial Record is the guarantee no step goes unchecked; assert the
  // keys so a step added to StepId can't quietly skip its gate.
  expect(Object.keys(validity)).toEqual([
    'basics',
    'credentials',
    'warehouses',
    'stages',
    'policies'
  ]);
  expect(Object.values(validity).every((v) => v.valid)).toBe(true);
});

test('stepDataValidity fails a step holding an incomplete item', () => {
  const blankPassword = { ...validCredential(), password: '' };
  expect(stepDataValidity(readyState({ credentials: [blankPassword] })).credentials.valid).toBe(
    false
  );
  // A Warehouse needs a name; initialWarehouse() has none.
  expect(stepDataValidity(readyState({ warehouses: [initialWarehouse()] })).warehouses.valid).toBe(
    false
  );
  // initialPolicy() with no stage name has nothing to target.
  expect(stepDataValidity(readyState({ policies: [initialPolicy()] })).policies.valid).toBe(false);
});

test('firstIncompleteStep passes a draft with valid data and skipped steps', () => {
  expect(firstIncompleteStep(readyState())).toBeUndefined();
  expect(firstIncompleteStep(readyState({ credentials: [validCredential()] }))).toBeUndefined();
});

// The case the Review gate exists for: a resumed draft arrives with its
// credential secrets stripped, and the sidebar can jump straight to Review
// without ever passing through the credentials step's own gate.
test('firstIncompleteStep catches a resumed credential whose secret was stripped', () => {
  const resumed = readyState({ credentials: [{ ...validCredential(), password: '' }] });
  expect(firstIncompleteStep(resumed)).toEqual({
    id: 'credentials',
    reason: 'Every credential needs a valid name, repository URL, and complete auth fields'
  });
});

test('firstIncompleteStep reports the earliest step, so the user is sent back once', () => {
  const state = readyState({
    basics: { name: 'Not A DNS Name', description: '' },
    warehouses: [initialWarehouse()]
  });
  expect(firstIncompleteStep(state)?.id).toBe('basics');
  // With basics fixed, the next unmet requirement surfaces.
  expect(firstIncompleteStep({ ...state, basics: { name: 'ok', description: '' } })?.id).toBe(
    'warehouses'
  );
});

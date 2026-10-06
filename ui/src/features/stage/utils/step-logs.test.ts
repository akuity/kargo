import { describe, expect, it } from 'vitest';

import { getStepLogs } from './step-logs';

describe('getStepLogs', () => {
  it('returns nothing when there is no output', () => {
    expect(getStepLogs()).toBeUndefined();
    expect(getStepLogs({})).toBeUndefined();
  });

  it('reads the singular log key too', () => {
    expect(getStepLogs({ log: 'a\nb\nc' })).toEqual(['a', 'b', 'c']);
  });

  it('reads an array of lines', () => {
    expect(getStepLogs({ lines: ['one', 'two'] })).toEqual(['one', 'two']);
  });

  it('ignores arrays that are empty or not all strings', () => {
    expect(getStepLogs({ lines: [] })).toBeUndefined();
    expect(getStepLogs({ lines: ['one', 2] })).toBeUndefined();
  });

  it('splits a multi-line string', () => {
    expect(getStepLogs({ logs: 'one\ntwo\nthree' })).toEqual(['one', 'two', 'three']);
  });

  it('normalizes CRLF and drops a single trailing newline', () => {
    expect(getStepLogs({ logs: 'one\r\ntwo\r\nthree\r\n' })).toEqual(['one', 'two', 'three']);
  });

  it('preserves interior blank lines and order', () => {
    expect(getStepLogs({ logs: 'one\n\nthree' })).toEqual(['one', '', 'three']);
  });

  it('ignores strings too short to read as a log', () => {
    expect(getStepLogs({ logs: 'abc123' })).toBeUndefined();
    expect(getStepLogs({ logs: 'abc123\ndef456' })).toBeUndefined();
  });

  it('ignores output that carries logs alongside other data', () => {
    expect(getStepLogs({ lines: ['x'], exitCode: 0 })).toBeUndefined();
    expect(getStepLogs({ logs: 'a\nb\nc', lines: ['x'] })).toBeUndefined();
  });

  it('ignores links, which the deep-link plugin renders separately', () => {
    expect(
      getStepLogs({
        lines: ['x'],
        links: [{ url: 'https://example.com', label: 'Build' }]
      })
    ).toEqual(['x']);
  });

  // Matching is by name, never by shape: ordinary outputs are often string
  // arrays or multi-line text.
  it('does not read log-shaped values under other keys', () => {
    expect(
      getStepLogs({
        number: 42,
        title: 'Promote v1.5.0',
        body: 'Notes\n\n- one\n- two',
        labels: ['release', 'automated'],
        url: 'https://example.com/issues/42'
      })
    ).toBeUndefined();

    expect(getStepLogs({ subnet_ids: ['subnet-a', 'subnet-b'] })).toBeUndefined();
  });

  // Generic names stay with the YAML view so already-shipped steps keep
  // rendering the way they do today.
  // Only the log-named keys are claimed. Console output under any other key --
  // including the `output` key the Text format uses for raw stdout -- keeps the
  // YAML view, so a step wanting the panel has to name it.
  it('leaves unlisted keys to the YAML view', () => {
    expect(getStepLogs({ output: 'a\nb\nc' })).toBeUndefined();
    expect(getStepLogs({ stdout: 'a\nb\nc' })).toBeUndefined();
    expect(getStepLogs({ stderr: 'a\nb\nc' })).toBeUndefined();
    expect(getStepLogs({ plan: 'a\nb\nc' })).toBeUndefined();
    expect(getStepLogs({ result: 'a\nb\nc' })).toBeUndefined();
  });
});

import { expect, test, describe } from 'vitest';

import { majorMinorVersion, releaseLabel } from './releases';
import { Release } from './types';

const cliBinaries = {
  darwin: { amd64: '', arm64: '' },
  linux: { amd64: '', arm64: '' },
  windows: { amd64: '', arm64: '' }
};

describe('majorMinorVersion', () => {
  test('parses standard semver with v prefix', () => {
    expect(majorMinorVersion('v1.2.3')).toBe('v1.2');
  });

  test('parses semver without v prefix', () => {
    expect(majorMinorVersion('1.2.3')).toBe('v1.2');
  });

  test('parses pre-release version', () => {
    expect(majorMinorVersion('v1.2.3-alpha.1')).toBe('v1.2');
  });

  test('parses build metadata version', () => {
    expect(majorMinorVersion('v1.2.3+build.42')).toBe('v1.2');
  });

  test('parses pre-release with build metadata', () => {
    expect(majorMinorVersion('v1.2.3-rc.1+build.1')).toBe('v1.2');
  });

  test('parses zero major version', () => {
    expect(majorMinorVersion('v0.9.0')).toBe('v0.9');
  });

  test('parses large version numbers', () => {
    expect(majorMinorVersion('v10.20.30')).toBe('v10.20');
  });

  test('returns original string for invalid semver', () => {
    expect(majorMinorVersion('not-a-version')).toBe('not-a-version');
  });

  test('returns original string for empty string', () => {
    expect(majorMinorVersion('')).toBe('');
  });

  test('returns original string for partial version', () => {
    expect(majorMinorVersion('v1.2')).toBe('v1.2');
  });

  test('parses -ak pre-release version', () => {
    expect(majorMinorVersion('v1.2.3-ak')).toBe('v1.2');
  });
});

describe('releaseLabel', () => {
  test('returns major.minor for non-latest release', () => {
    const release: Release = { version: 'v1.5.0', cliBinaries };
    expect(releaseLabel(release)).toBe('v1.5');
  });

  test('appends (latest) for latest release', () => {
    const release: Release = { version: 'v1.5.0', latest: true, cliBinaries };
    expect(releaseLabel(release)).toBe('v1.5 (latest)');
  });

  test('latest: false does not append (latest)', () => {
    const release: Release = { version: 'v1.5.0', latest: false, cliBinaries };
    expect(releaseLabel(release)).toBe('v1.5');
  });

  test('handles pre-release version', () => {
    const release: Release = { version: 'v2.0.0-rc.1', latest: false, cliBinaries };
    expect(releaseLabel(release)).toBe('v2.0');
  });

  test('handles pre-release latest', () => {
    const release: Release = { version: 'v2.0.0-rc.1', latest: true, cliBinaries };
    expect(releaseLabel(release)).toBe('v2.0 (latest)');
  });

  test('handles invalid version gracefully', () => {
    const release: Release = { version: 'invalid', cliBinaries };
    expect(releaseLabel(release)).toBe('invalid');
  });
});

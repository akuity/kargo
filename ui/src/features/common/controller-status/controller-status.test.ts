import { describe, expect, test } from 'vitest';

import { Heartbeat, HeartbeatStatus } from '../../../gen/api/v2/models';

import { ControllerStatus, deadControllerReason, getControllerStatus } from './controller-status';

describe('getControllerStatus()', () => {
  const heartbeats: Record<string, Heartbeat> = {
    default: { status: HeartbeatStatus.StatusAlive },
    'us-east': { status: HeartbeatStatus.StatusAlive },
    'eu-west': { status: HeartbeatStatus.StatusDead }
  };

  const testCases: {
    name: string;
    shard?: string;
    heartbeats?: Record<string, Heartbeat>;
    defaultControllerName?: string;
    expected: ControllerStatus;
  }[] = [
    {
      name: 'heartbeats not loaded yet',
      shard: 'eu-west',
      expected: { controllerName: 'eu-west', isControllerDead: false }
    },
    {
      name: 'no shard, live default controller',
      heartbeats,
      defaultControllerName: 'default',
      expected: { controllerName: 'default', isControllerDead: false }
    },
    {
      name: 'no shard, no default controller',
      heartbeats,
      expected: { controllerName: '', isControllerDead: true }
    },
    {
      name: 'shard with a live controller',
      shard: 'us-east',
      heartbeats,
      defaultControllerName: 'default',
      expected: { controllerName: 'us-east', isControllerDead: false }
    },
    {
      name: 'shard with a dead controller',
      shard: 'eu-west',
      heartbeats,
      defaultControllerName: 'default',
      expected: { controllerName: 'eu-west', isControllerDead: true }
    },
    {
      name: 'shard no controller has ever reported',
      shard: 'typo',
      heartbeats,
      defaultControllerName: 'default',
      expected: { controllerName: 'typo', isControllerDead: true }
    }
  ];

  for (const testCase of testCases) {
    test(testCase.name, () => {
      expect(
        getControllerStatus(testCase.shard, testCase.heartbeats, testCase.defaultControllerName)
      ).toEqual(testCase.expected);
    });
  }
});

describe('deadControllerReason()', () => {
  test('names the controller', () => {
    expect(deadControllerReason('eu-west')).toBe("Controller 'eu-west' is dead or nonexistent");
  });

  test('falls back to the default controller', () => {
    expect(deadControllerReason('')).toBe('The default controller is dead or nonexistent');
  });
});

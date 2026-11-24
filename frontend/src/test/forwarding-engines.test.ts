import { describe, expect, it } from 'vitest';

import type { ForwardingEngine } from '@/generated/zod';
import {
  isBuiltIn,
  isForwarder,
  modeAfterEngineChange,
  modesOf,
  requiredSettings,
  settingsAsOptions,
  transportAfterChange,
  transportsFor,
} from '@/pages/tunnels/forwardingEngines';

const tr = (id: string, reverse: boolean, direct: boolean) => ({
  id,
  label: id.toUpperCase(),
  reverse,
  direct,
});

const rathole: ForwardingEngine = {
  id: 'rathole',
  label: 'Rathole',
  pinned: 'v0.5.0',
  transports: [tr('tcp', true, false), tr('noise', true, false)],
  options: [],
};
const gost: ForwardingEngine = {
  id: 'gost',
  label: 'GOST',
  pinned: 'v3.3.3',
  transports: [tr('tls', true, true), tr('ws', true, true)],
  options: [],
};
const singbox: ForwardingEngine = {
  id: 'singbox',
  label: 'sing-box',
  pinned: 'v1.14.4',
  transports: [tr('tcp-tls', false, true)],
  options: [],
};
const mixed: ForwardingEngine = {
  id: 'kariz',
  label: 'Kariz',
  pinned: 'v2.8.1',
  transports: [tr('a', true, false), tr('b', false, true)],
  options: [
    { key: 'profile', label: 'Profile', choices: ['balanced', 'gaming'], default: 'balanced' },
  ],
};

const carrier: ForwardingEngine = {
  id: 'carrier',
  label: 'Kernel carriers',
  pinned: 'built in',
  transports: [tr('gre', true, true)],
  options: [
    { key: 'dialerIp', label: 'Public IPv4 address', choices: [], default: '', kind: 'ip' },
  ],
};

describe('forwarding engines', () => {
  it('lists the modes an engine can run, reverse first', () => {
    expect(modesOf(rathole)).toEqual(['reverse']);
    expect(modesOf(singbox)).toEqual(['direct']);
    expect(modesOf(gost)).toEqual(['reverse', 'direct']);
  });

  it('keeps the mode when the new engine can run it and otherwise takes the engine mode', () => {
    expect(modeAfterEngineChange(gost, 'direct')).toBe('direct');
    expect(modeAfterEngineChange(rathole, 'direct')).toBe('reverse');
    expect(modeAfterEngineChange(singbox, 'reverse')).toBe('direct');
  });

  it('offers only the transports that run in the chosen mode', () => {
    expect(transportsFor(mixed, 'reverse').map((o) => o.value)).toEqual(['a']);
    expect(transportsFor(mixed, 'direct').map((o) => o.value)).toEqual(['b']);
  });

  it('keeps a transport that still runs and falls back to the first one that does', () => {
    expect(transportAfterChange(gost, 'reverse', 'ws')).toBe('ws');
    expect(transportAfterChange(mixed, 'direct', 'a')).toBe('b');
    expect(transportAfterChange(rathole, 'reverse', undefined)).toBe('tcp');
  });

  it('turns an engine setting into a choice option for the shared form', () => {
    expect(settingsAsOptions(mixed)).toMatchObject([
      {
        key: 'profile',
        kind: 'choice',
        role: 'both',
        mode: 'both',
        choices: ['balanced', 'gaming'],
      },
    ]);
    expect(settingsAsOptions(undefined)).toEqual([]);
  });

  it('turns an address setting into a text option the tunnel cannot be made without', () => {
    const options = settingsAsOptions(carrier);
    expect(options).toMatchObject([
      { key: 'dialerIp', kind: 'text', hintEn: 'Public IPv4 address' },
    ]);
    expect(requiredSettings(options)).toEqual(['dialerIp']);
    expect(requiredSettings(settingsAsOptions(mixed))).toEqual([]);
  });

  it('knows which engine is part of the panel and has nothing to install', () => {
    expect(isBuiltIn('carrier')).toBe(true);
    expect(isBuiltIn('gost')).toBe(false);
    expect(isBuiltIn(undefined)).toBe(false);
  });

  it('tells a forwarding engine id from the two built-in engines', () => {
    expect(isForwarder('gost')).toBe(true);
    expect(isForwarder('xraycore')).toBe(true);
    expect(isForwarder('backpack')).toBe(false);
    expect(isForwarder('rakhsh')).toBe(false);
    expect(isForwarder('toString')).toBe(false);
    expect(isForwarder(undefined)).toBe(false);
  });
});

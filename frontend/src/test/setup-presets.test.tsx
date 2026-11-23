import { describe, expect, it } from 'vitest';

import { buildPresetInbound } from '@/pages/setup/presets';
import { InboundFormSchema } from '@/schemas/forms/inbound-form';

const keys = { privateKey: 'priv-key', publicKey: 'pub-key' };

describe('first-run connection presets', () => {
  it('builds a valid VLESS Reality inbound on 443 with the first client', () => {
    const values = buildPresetInbound('reality', { email: 'sara.k', keys });
    expect(InboundFormSchema.safeParse(values).success).toBe(true);
    expect(values.protocol).toBe('vless');
    expect(values.port).toBe(443);
    const stream = values.streamSettings as Record<string, unknown>;
    expect(stream.security).toBe('reality');
    const reality = stream.realitySettings as {
      privateKey: string;
      target: string;
      serverNames: string[];
      shortIds: string[];
    };
    expect(reality.privateKey).toBe('priv-key');
    expect(reality.target).toBe('www.speedtest.net:443');
    expect(reality.serverNames).toEqual(['www.speedtest.net']);
    expect(reality.shortIds[0]).toMatch(/^[0-9a-f]{2,16}$/);
    const clients = (values.settings as { clients: { email: string; flow: string }[] }).clients;
    expect(clients.map((c) => [c.email, c.flow])).toEqual([['sara.k', 'xtls-rprx-vision']]);
  });

  it('builds a valid Trojan over WebSocket inbound for a CDN, without TLS on the server', () => {
    const values = buildPresetInbound('cdn', { email: 'sara.k' });
    expect(InboundFormSchema.safeParse(values).success).toBe(true);
    expect(values.protocol).toBe('trojan');
    expect(values.port).toBe(8080);
    const stream = values.streamSettings as {
      network: string;
      security: string;
      wsSettings: { path: string };
    };
    expect([stream.network, stream.security]).toEqual(['ws', 'none']);
    expect(stream.wsSettings.path).toMatch(/^\/[a-zA-Z0-9]{8,}$/);
  });
});

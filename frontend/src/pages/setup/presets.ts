import { rawInboundToFormValues } from '@/lib/xray/inbound-form-adapter';
import {
  createDefaultInboundSettings,
  createDefaultTrojanClient,
  createDefaultVlessClient,
} from '@/lib/xray/inbound-defaults';
import { SniffingSchema } from '@/schemas/primitives/sniffing';
import { RealityStreamSettingsSchema } from '@/schemas/protocols/security/reality';
import { TcpStreamSettingsSchema } from '@/schemas/protocols/stream/tcp';
import { WsStreamSettingsSchema } from '@/schemas/protocols/stream/ws';
import type { InboundFormValues } from '@/schemas/forms/inbound-form';
import { RandomUtil } from '@/utils';

export type Preset = 'reality' | 'cdn';

// A site that serves TLS 1.3 and HTTP/2 from Iran and abroad, so Reality can borrow its handshake.
const REALITY_SITE = 'www.speedtest.net';

interface PresetInput {
  email: string;
  totalGB?: number;
  expiryTime?: number;
  keys?: { privateKey: string; publicKey: string };
}

// The inbound the first-run wizard creates for a preset, as inbound form values.
export function buildPresetInbound(preset: Preset, input: PresetInput): InboundFormValues {
  const client = {
    email: input.email,
    totalGB: input.totalGB ?? 0,
    expiryTime: input.expiryTime ?? 0,
  };
  const base = {
    sniffing: SniffingSchema.parse({}),
    listen: '',
    tag: '',
    enable: true,
    trafficReset: 'never',
  };
  if (preset === 'reality') {
    const reality = RealityStreamSettingsSchema.parse({}) as Record<string, unknown>;
    reality.target = `${REALITY_SITE}:443`;
    reality.serverNames = [REALITY_SITE];
    reality.privateKey = input.keys?.privateKey ?? '';
    reality.shortIds = RandomUtil.randomShortIds()
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    reality.settings = {
      ...(reality.settings as Record<string, unknown>),
      publicKey: input.keys?.publicKey ?? '',
      spiderX: `/${RandomUtil.randomSeq(15)}`,
    };
    const settings = createDefaultInboundSettings('vless') as Record<string, unknown>;
    settings.clients = [createDefaultVlessClient({ ...client, flow: 'xtls-rprx-vision' })];
    return rawInboundToFormValues({
      ...base,
      protocol: 'vless',
      port: 443,
      settings,
      streamSettings: {
        network: 'tcp',
        security: 'reality',
        tcpSettings: TcpStreamSettingsSchema.parse({ header: { type: 'none' } }),
        realitySettings: reality,
      },
    } as never);
  }
  const settings = createDefaultInboundSettings('trojan') as Record<string, unknown>;
  settings.clients = [createDefaultTrojanClient(client)];
  const ws = WsStreamSettingsSchema.parse({}) as Record<string, unknown>;
  ws.path = `/${RandomUtil.randomSeq(12)}`;
  return rawInboundToFormValues({
    ...base,
    protocol: 'trojan',
    port: 8080,
    settings,
    streamSettings: { network: 'ws', security: 'none', wsSettings: ws },
  } as never);
}

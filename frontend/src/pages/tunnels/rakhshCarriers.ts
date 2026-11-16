// Mirrors rakhsh.Carriers and rakhsh.RawCarriers; the panel refuses a carrier this list lacks.
const RAW_CARRIERS = [
  'faketcp',
  'icmp',
  'spoof',
  'l2tcp',
  'shabdiz-gre',
  'shabdiz-gre6',
  'shabdiz-esp',
  'shabdiz-ipip',
  'shabdiz-6to4',
  'shabdiz-etherip',
  'shabdiz-l2tp',
  'shabdiz-mpls',
  'shabdiz-l2',
];
const STREAM_CARRIERS = [
  'tcp',
  'tls',
  'ws',
  'wss',
  'quic',
  'quicv2',
  'kcp',
  'shabdiz-udp',
  'shabdiz-tcp',
];

export const RAKHSH_CARRIERS = [...STREAM_CARRIERS, ...RAW_CARRIERS].map((value) => ({
  value,
  label: value,
}));

// Mirrors rakhsh's linkOverrides: the settings one link may override.
export const LINK_OVERRIDE_KEYS = [
  'sni',
  'ws_path',
  'ws_credential',
  'decoy',
  'edge_ip',
  'tls_fingerprint',
  'tls_frag',
  'cert_path',
  'key_path',
  'acme_domain',
  'acme_email',
  'acme_dir',
  'acme_port',
  'acme_staging',
  'quic_mask',
  'quic_version',
  'reliability',
  'faketcp_header',
  'pace_burst',
  'pace_gap_us',
  'pace_pps',
  'pace_auto',
];

export const isRawCarrier = (carrier: string | undefined) =>
  carrier !== undefined && RAW_CARRIERS.includes(carrier);

// Raw carriers forge packets to the peer, so the engine needs its address on both ends.
export const needsPeerIp = (carriers: (string | undefined)[]) => carriers.some(isRawCarrier);

// Mirrors the BackPack transports the panel accepts.
export const BACKPACK_TRANSPORTS = [
  'tcp',
  'tcpmux',
  'stealth',
  'ws',
  'wss',
  'wsmux',
  'wssmux',
  'kcp',
  'quic',
  'udp',
  'xdi',
  'pck',
].map((value) => ({ value, label: value }));

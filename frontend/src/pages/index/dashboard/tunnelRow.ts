export interface NodeRef {
  id: number;
  name: string;
  country?: string;
}

export interface TunnelLike {
  name: string;
  engine: string;
  mode: string;
  nodeId?: number | null;
  options?: Record<string, unknown> | null;
}

// BackPack's direct mode is its layer-3 tunnel; Rakhsh is layer 3 only with an l3_addr.
export function tunnelRow(tunnel: TunnelLike, nodes: readonly NodeRef[]) {
  const node = tunnel.nodeId ? nodes.find((n) => n.id === tunnel.nodeId) : undefined;
  const layer3 =
    tunnel.engine === 'backpack'
      ? tunnel.mode === 'direct'
      : tunnel.engine === 'rakhsh' && tunnel.options?.l3_addr !== undefined;
  return {
    to: node?.name ?? tunnel.name,
    country: node?.country || undefined,
    mode: tunnel.mode === 'direct' ? 'direct' : 'reverse',
    layer3,
  };
}

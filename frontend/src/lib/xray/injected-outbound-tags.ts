// Outbounds the panel adds at runtime (subscriptions, tunnels, Tor exits); not stored in the template.
export function injectedOutboundTags(payload: {
  subscriptionOutboundTags?: unknown;
  tunnelOutbounds?: unknown;
  torOutbounds?: unknown;
}): string[] {
  const tags: string[] = [];
  if (Array.isArray(payload.subscriptionOutboundTags)) {
    for (const t of payload.subscriptionOutboundTags) if (typeof t === 'string' && t) tags.push(t);
  }
  for (const list of [payload.tunnelOutbounds, payload.torOutbounds]) {
    if (!Array.isArray(list)) continue;
    for (const o of list) {
      const tag = (o as { tag?: unknown } | null)?.tag;
      if (typeof tag === 'string' && tag) tags.push(tag);
    }
  }
  return tags;
}

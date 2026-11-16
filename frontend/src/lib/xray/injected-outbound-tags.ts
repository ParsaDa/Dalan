// Outbounds the panel adds at runtime (subscriptions, tunnels); not stored in the template.
export function injectedOutboundTags(payload: {
  subscriptionOutboundTags?: unknown;
  tunnelOutbounds?: unknown;
}): string[] {
  const tags: string[] = [];
  if (Array.isArray(payload.subscriptionOutboundTags)) {
    for (const t of payload.subscriptionOutboundTags) if (typeof t === 'string' && t) tags.push(t);
  }
  if (Array.isArray(payload.tunnelOutbounds)) {
    for (const o of payload.tunnelOutbounds) {
      const tag = (o as { tag?: unknown } | null)?.tag;
      if (typeof tag === 'string' && tag) tags.push(tag);
    }
  }
  return tags;
}

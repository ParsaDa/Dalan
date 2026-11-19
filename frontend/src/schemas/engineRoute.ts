import { z } from 'zod';

export const EngineRouteSchema = z.object({
  via: z.enum(['panel', 'direct', 'outbound', 'link']),
  value: z.string(),
});
export type EngineRoute = z.infer<typeof EngineRouteSchema>;

export const PANEL_ROUTE: EngineRoute = { via: 'panel', value: '' };

// An outbound needs its tag and a link needs an address; the other two ways carry nothing.
export function engineRouteReady(route: EngineRoute): boolean {
  if (route.via === 'outbound' || route.via === 'link') return route.value.trim() !== '';
  return true;
}

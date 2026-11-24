import { z } from 'zod';

export const SpeedtestResultSchema = z.object({
  server: z.object({
    id: z.string(),
    name: z.string(),
    sponsor: z.string(),
    country: z.string(),
    host: z.string(),
    distance: z.number(),
  }),
  pingMs: z.number(),
  jitterMs: z.number(),
  downMbps: z.number(),
  upMbps: z.number(),
  at: z.number(),
});
export type SpeedtestResult = z.infer<typeof SpeedtestResultSchema>;

export const SpeedtestStatusSchema = z.object({
  running: z.boolean(),
  phase: z.string(),
  progress: z.number(),
  mbps: z.number(),
  pingMs: z.number(),
  result: SpeedtestResultSchema.nullable(),
  error: z.string(),
});
export type SpeedtestStatus = z.infer<typeof SpeedtestStatusSchema>;

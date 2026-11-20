import { z } from 'zod';

const requiredText = z.string().trim().min(1, 'pages.enrolments.required');

// The node's own tunnel is named after it, and the management one "node-<name>"; a tunnel name holds 31 characters.
export const EnrolmentFormSchema = z.object({
  name: z
    .string()
    .trim()
    .regex(/^[a-z0-9][a-z0-9-]{0,25}$/, 'pages.enrolments.nameInvalid'),
  panelUrl: requiredText,
  address: requiredText,
  port: z.number().int().min(1).max(65535),
  engine: z.enum(['backpack', 'rakhsh']),
  transport: requiredText,
  tunnelId: z.number().int().min(0),
});

export type EnrolmentFormValues = z.infer<typeof EnrolmentFormSchema>;

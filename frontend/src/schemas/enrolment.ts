import { z } from 'zod';

const requiredText = z.string().trim().min(1, 'pages.enrolments.required');

// REALITY imitates a real site, so a tunnel secured with it cannot be created without one.
export const RealityServerNameSchema = requiredText;

// The management tunnel is named "node-<name>", and a tunnel name holds 31 characters.
export const EnrolmentFormSchema = z.object({
  name: z
    .string()
    .trim()
    .regex(/^[a-z0-9][a-z0-9-]{0,25}$/, 'pages.enrolments.nameInvalid'),
  panelUrl: requiredText,
  address: requiredText,
  port: z.number().int().min(1).max(65535),
  security: z.enum(['reality', 'none']),
  serverName: z.string().trim(),
});

export type EnrolmentFormValues = z.infer<typeof EnrolmentFormSchema>;

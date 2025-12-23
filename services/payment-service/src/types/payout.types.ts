import { z } from 'zod';

export const payoutSchema = z.object({
  id: z.string(),
  amount: z.coerce.number().positive(),
  description: z.string().min(1).max(255),
  status: z.enum(['pending', 'scanned', 'completed']).default('pending'),
  user_id: z.string(),
  bank_code: z.string().nullable().optional(),
  account_number: z.string().nullable().optional(),
  created_at: z.date(),
  updated_at: z.date(),
  deleted_at: z.date().nullable().optional(),
  completed_at: z.date().nullable().optional(),
});

const payoutStatusMap: Record<z.infer<typeof payoutSchema>['status'], string> =
  {
    pending: 'Chờ chi trả',
    scanned: 'Đã quét',
    completed: 'Đã chi trả',
  };

export const payoutViewSchema = payoutSchema.transform((p) => ({
  ...p,
  status: {
    code: p.status,
    label: payoutStatusMap[p.status] ?? p.status,
  },
}));

export type PayoutDto = z.infer<typeof payoutSchema>;
export type PayoutViewDto = z.infer<typeof payoutViewSchema>;

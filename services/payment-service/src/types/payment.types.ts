import { z } from 'zod';

export const createPaymentSchema = z.object({
  id: z.string(),
  amount: z.coerce.number().positive(),
  description: z.string().min(1).max(255),
  user_id: z.string(),
  order_code: z.string().max(255).nullable().optional(),
  status: z
    .enum([
      'pending',
      'completed',
      'failed',
      'refunded',
      'cancelled',
      'expired',
    ])
    .optional(),
});

export const paymentSchema = z.object({
  id: z.string(),
  amount: z.coerce.number().positive(),
  description: z.string().min(1).max(255),
  status: z
    .enum([
      'pending',
      'completed',
      'failed',
      'refunded',
      'cancelled',
      'expired',
    ])
    .default('pending'),
  user_id: z.string(),
  checkout_url: z.string().max(255).nullable().optional(),
  type: z.string().max(50).nullable().optional(),
  order_code: z.string().max(255).nullable().optional(),
  created_at: z.date(),
  updated_at: z.date(),
  deleted_at: z.date().nullable().optional(),
  paid_at: z.date().nullable().optional(),
  expired_at: z.date().nullable().optional(),
});

const statusMap: Record<z.infer<typeof paymentSchema>['status'], string> = {
  pending: 'Chờ thanh toán',
  completed: 'Đã thanh toán',
  failed: 'Thất bại',
  refunded: 'Đã hoàn tiền',
  cancelled: 'Đã hủy',
  expired: 'Đã hết hạn',
};

export const paymentViewSchema = paymentSchema.transform((p) => {
  // Nếu expired_at < now và status là 'pending', map thành 'expired'
  const now = new Date();
  const effectiveStatus =
    p.expired_at && p.expired_at < now && p.status === 'pending'
      ? 'expired'
      : p.status;
  return {
    ...p,
    status: statusMap[effectiveStatus] ?? effectiveStatus,
  };
});

export const updatePaymentSchema = createPaymentSchema.partial();

export const findOrdersResponseSchema = z.object({
  message: z.string(),
  code: z.number(),
  data: z.array(paymentSchema),
  total_pages: z.number(),
  current_page: z.number(),
  total_items: z.number(),
  previous: z.boolean().nullable().optional(),
  next: z.boolean().nullable().optional(),
});

export type CreatePaymentDto = z.infer<typeof createPaymentSchema>;
export type UpdatePaymentDto = z.infer<typeof updatePaymentSchema>;
export type PaymentDto = z.infer<typeof paymentSchema>;
export type FindOrdersResponseDto = z.infer<typeof findOrdersResponseSchema>;
export type PaymentViewDto = z.infer<typeof paymentViewSchema>;

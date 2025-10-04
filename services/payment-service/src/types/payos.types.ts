import { z } from 'zod';

export const paymentLinkSchema = z.object({
  bin: z.string().nullable().optional(),
  checkout_url: z.string().url().nullable().optional(),
  account_number: z.string().nullable().optional(),
  account_name: z.string().nullable().optional(),
  amount: z.coerce.number().nullable().optional(),
  description: z.string().nullable().optional(),
  order_code: z.union([z.string(), z.number()]).nullable().optional(),
  qr_code: z.string().nullable().optional(),
  expired_at: z
    .union([z.number(), z.date()])
    .nullable()
    .optional()
    .transform((val) => {
      if (val === null || val === undefined) return null;
      if (typeof val === 'number') return new Date(val * 1000);
      return val;
    }),
});

export type PaymentLinkDto = z.infer<typeof paymentLinkSchema>;

export const createPaymentLinkSchema = z.object({
  order_code: z.number().optional(),
  amount: z.coerce.number(),
  description: z.string(),
  return_url: z.string().url().optional(),
  cancel_url: z.string().url().optional(),
});

export type CreatePaymentLinkData = z.infer<typeof createPaymentLinkSchema> & {
  expired_at: Date;
};

export const paymentLinkResponseSchema = z.object({
  bin: z.string().nullable().optional(),
  checkout_url: z.string().url().nullable().optional(),
  account_number: z.string().nullable().optional(),
  account_name: z.string().nullable().optional(),
  amount: z.coerce.number().nullable().optional(),
  description: z.string().nullable().optional(),
  order_code: z.number().nullable().optional(),
  qr_code: z.string().nullable().optional(),
  expired_at: z
    .union([z.number(), z.date()])
    .nullable()
    .optional()
    .transform((val) => {
      if (val === null || val === undefined) return null;
      if (typeof val === 'number') return new Date(val * 1000);
      return val;
    }),
});

export type PaymentLinkResponse = z.infer<typeof paymentLinkResponseSchema>;

export const serviceResponseSchema = <T extends z.ZodTypeAny>(dataSchema: T) =>
  z.object({
    error: z.number(),
    message: z.string(),
    data: dataSchema.nullable(),
  });

export const servicePaymentLinkResponseSchema = serviceResponseSchema(
  paymentLinkResponseSchema,
);

export const servicePaymentLinkDtoResponseSchema =
  serviceResponseSchema(paymentLinkSchema);

export type ServicePaymentLinkResponse = z.infer<
  typeof servicePaymentLinkResponseSchema
>;

export type ServicePaymentLinkDtoResponse = z.infer<
  typeof servicePaymentLinkDtoResponseSchema
>;

// Schema cho payload webhook từ PayOS (dùng camelCase cho data, match payload thực tế)
export const webhookPayloadSchema = z.object({
  code: z.string(), // "00" = success
  desc: z.string(),
  success: z.boolean().optional(), // Thêm success (optional)
  data: z.object({
    accountNumber: z.string(),
    amount: z.number(),
    description: z.string(),
    reference: z.string(),
    transactionDateTime: z.string(),
    virtualAccountNumber: z.string(),
    counterAccountBankId: z.string().nullable(),
    counterAccountBankName: z.string().nullable(),
    counterAccountName: z.string().nullable(),
    counterAccountNumber: z.string().nullable(),
    virtualAccountName: z.string().nullable(),
    currency: z.string().optional(),
    orderCode: z.number(),
    paymentLinkId: z.string(),
    code: z.string(),
    desc: z.string(),
  }),
  signature: z.string(),
});

import { z } from 'zod';

export const paymentLinkSchema = z.object({
  bin: z.string().nullable().optional(),
  checkout_url: z.string().url().nullable().optional(),
  account_number: z.string().nullable().optional(),
  account_name: z.string().nullable().optional(),
  amount: z.number().nullable().optional(),
  description: z.string().nullable().optional(),
  order_code: z.union([z.string(), z.number()]).nullable().optional(),
  qr_code: z.string().nullable().optional(),
});

export type PaymentLinkDto = z.infer<typeof paymentLinkSchema>;

export const createPaymentLinkSchema = z.object({
  order_code: z.number().optional(),
  amount: z.number(),
  description: z.string(),
  return_url: z.string().url().optional(),
  cancel_url: z.string().url().optional(),
});

export type CreatePaymentLinkData = z.infer<typeof createPaymentLinkSchema>;

export const paymentLinkResponseSchema = z.object({
  bin: z.string().nullable().optional(),
  checkout_url: z.string().url().nullable().optional(),
  account_number: z.string().nullable().optional(),
  account_name: z.string().nullable().optional(),
  amount: z.number().nullable().optional(),
  description: z.string().nullable().optional(),
  order_code: z.number().nullable().optional(),
  qr_code: z.string().nullable().optional(),
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

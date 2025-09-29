import { z } from 'zod';

export const paymentLinkSchema = z.object({
  bin: z.string().nullable().optional(),
  checkoutUrl: z.string().url().nullable().optional(),
  accountNumber: z.string().nullable().optional(),
  accountName: z.string().nullable().optional(),
  amount: z.number().nullable().optional(),
  description: z.string().nullable().optional(),
  orderCode: z.union([z.string(), z.number()]).nullable().optional(),
  qrCode: z.string().nullable().optional(),
});

export type PaymentLinkDto = z.infer<typeof paymentLinkSchema>;

export const createPaymentLinkSchema = z.object({
  orderCode: z.number(),
  amount: z.number(),
  description: z.string(),
  returnUrl: z.url(),
  cancelUrl: z.url(),
});

export type CreatePaymentLinkData = z.infer<typeof createPaymentLinkSchema>;

export const paymentLinkResponseSchema = z.object({
  bin: z.string().nullable().optional(),
  checkoutUrl: z.url().nullable().optional(),
  accountNumber: z.string().nullable().optional(),
  accountName: z.string().nullable().optional(),
  amount: z.number().nullable().optional(),
  description: z.string().nullable().optional(),
  orderCode: z.number().nullable().optional(),
  qrCode: z.string().nullable().optional(),
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

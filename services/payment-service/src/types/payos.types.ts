import { z } from 'zod';

export const paymentLinkSchema = z.object({
  bin: z.string().nullable().optional(),
  checkout_url: z.url().nullable().optional(),
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
  type: z.string().optional(),
  items: z
    .array(
      z.object({
        item_id: z.string().optional(),
        name: z.string(),
        price: z.coerce.number().positive(),
        quantity: z.coerce.number().int().positive().default(1),
      }),
    )
    .optional(),
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
  items: z
    .array(
      z.object({
        name: z.string(),
        price: z.coerce.number().positive(),
        quantity: z.coerce.number().int().positive().default(1),
      }),
    )
    .optional(),
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

export const webhookPayloadSchema = z.object({
  code: z.string(),
  desc: z.string(),
  success: z.boolean().optional(),
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

// Payout types
export const payoutTransactionSchema = z.object({
  id: z.string(),
  referenceId: z.string(),
  amount: z.number(),
  description: z.string(),
  toBin: z.string(),
  toAccountNumber: z.string(),
  toAccountName: z.string().nullable(),
  reference: z.string().nullable(),
  transactionDatetime: z.string().nullable(),
  errorMessage: z.string().nullable(),
  errorCode: z.string().nullable(),
  state: z.enum(['PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED', 'CANCELLED']),
});

export const payoutSchema = z.object({
  id: z.string(),
  referenceId: z.string(),
  transactions: z.array(payoutTransactionSchema),
  category: z.array(z.string()),
  approvalState: z.enum(['PENDING', 'APPROVED', 'REJECTED']),
  createdAt: z.string(),
});

export const createPayoutDataSchema = z.object({
  referenceId: z.string().optional(),
  amount: z.number(),
  description: z.string(),
  toBin: z.string(),
  toAccountNumber: z.string(),
  category: z.array(z.string()).optional(),
});

export const createBatchPayoutDataSchema = z.object({
  referenceId: z.string().optional(),
  category: z.array(z.string()).optional(),
  validateDestination: z.boolean().optional(),
  payouts: z.array(
    z.object({
      referenceId: z.string().optional(),
      amount: z.number(),
      description: z.string(),
      toBin: z.string(),
      toAccountNumber: z.string(),
    }),
  ),
});

export const payoutAccountBalanceSchema = z.object({
  accountNumber: z.string(),
  accountName: z.string(),
  currency: z.string(),
  balance: z.string(),
});

export const estimateCreditSchema = z.object({
  estimateCredit: z.number(),
});

export type PayoutTransactionDto = z.infer<typeof payoutTransactionSchema>;
export type PayoutDto = z.infer<typeof payoutSchema>;
export type CreatePayoutData = z.infer<typeof createPayoutDataSchema>;
export type CreateBatchPayoutData = z.infer<typeof createBatchPayoutDataSchema>;
export type PayoutAccountBalanceDto = z.infer<
  typeof payoutAccountBalanceSchema
>;
export type EstimateCreditDto = z.infer<typeof estimateCreditSchema>;

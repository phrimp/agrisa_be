import {
  PaymentLinkDto,
  CreatePaymentLinkData,
  PaymentLinkResponse,
  PayoutDto,
  CreatePayoutData,
  CreateBatchPayoutData,
  PayoutAccountBalanceDto,
  EstimateCreditDto,
} from '../types/payos.types';

type ServiceResponse<T = unknown> = {
  error: number;
  message: string;
  data: T | null;
};

export interface PayosService {
  createPaymentLink(
    data: CreatePaymentLinkData,
  ): Promise<ServiceResponse<PaymentLinkResponse>>;
  getPaymentLinkInfo(
    order_id: string,
  ): Promise<ServiceResponse<PaymentLinkDto>>;
  cancelPaymentLink(
    order_id: string,
    cancellation_reason: string,
  ): Promise<ServiceResponse<PaymentLinkDto | Record<string, unknown>>>;
  verifyPaymentWebhookData(
    webhookData: unknown,
  ): PaymentLinkDto | Record<string, unknown>;
  confirmWebhook(webhook_url: string): Promise<ServiceResponse<null>>;
  getExpiredDuration(): string;
  getOrderCodeLength(): number;

  // Payout methods
  createPayout(data: CreatePayoutData): Promise<ServiceResponse<PayoutDto>>;
  createBatchPayout(
    data: CreateBatchPayoutData,
  ): Promise<ServiceResponse<PayoutDto>>;
  getPayout(payoutId: string): Promise<ServiceResponse<PayoutDto>>;
  getPayouts(options?: {
    limit?: number;
    offset?: number;
    referenceId?: string;
    approvalState?: string;
    category?: string;
    fromDate?: string;
    toDate?: string;
  }): Promise<ServiceResponse<{ payouts: PayoutDto[]; pagination: any }>>;
  estimatePayoutCredit(
    data: CreateBatchPayoutData,
  ): Promise<ServiceResponse<EstimateCreditDto>>;
  getPayoutAccountBalance(): Promise<ServiceResponse<PayoutAccountBalanceDto>>;
}

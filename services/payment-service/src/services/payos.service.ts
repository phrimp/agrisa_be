import {
  PaymentLinkDto,
  CreatePaymentLinkData,
  PaymentLinkResponse,
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
}

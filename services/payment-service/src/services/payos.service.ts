import {
  PaymentLinkDto,
  CreatePaymentLinkData,
  PaymentLinkResponse,
} from '../entities/payos.entity';

type ServiceResponse<T = unknown> = {
  error: number;
  message: string;
  data: T | null;
};

export interface PayosService {
  createPaymentLink(
    data: CreatePaymentLinkData,
  ): Promise<ServiceResponse<PaymentLinkResponse>>;
  getPaymentLinkInfo(orderId: string): Promise<ServiceResponse<PaymentLinkDto>>;
  cancelPaymentLink(
    orderId: string,
    cancellationReason: string,
  ): Promise<ServiceResponse<PaymentLinkDto | Record<string, unknown>>>;
  verifyPaymentWebhookData(
    webhookData: unknown,
  ): PaymentLinkDto | Record<string, unknown>;
  confirmWebhook(webhookUrl: string): Promise<ServiceResponse<null>>;
}

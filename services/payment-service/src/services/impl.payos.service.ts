import { Injectable, Logger } from '@nestjs/common';
import { payOS } from '../libs/payos.config';
import {
  paymentLinkSchema,
  PaymentLinkDto,
  CreatePaymentLinkData,
  PaymentLinkResponse,
} from '../entities/payos.entity';
import { PayosService } from './payos.service';
type PayOSClient = {
  paymentRequests: {
    create: (data: CreatePaymentLinkData) => Promise<unknown>;
    get: (orderCode: string | number) => Promise<unknown>;
    cancel: (
      orderCode: string | number,
      opts: { cancellationReason: string },
    ) => Promise<unknown>;
  };
  webhooks: {
    verify: (webhookData: unknown) => unknown;
    confirm: (webhookUrl: string) => Promise<void>;
  };
};

type ServiceResponse<T = unknown> = {
  error: number;
  message: string;
  data: T | null;
};

@Injectable()
export class ImplPayosService implements PayosService {
  private readonly logger = new Logger(ImplPayosService.name);
  private readonly payOS: PayOSClient;

  constructor() {
    this.payOS = payOS as unknown as PayOSClient;
  }

  async createPaymentLink(
    data: CreatePaymentLinkData,
  ): Promise<ServiceResponse<PaymentLinkResponse>> {
    try {
      const raw = await this.payOS.paymentRequests.create(data);
      const parsed = paymentLinkSchema.safeParse(raw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi createPaymentLink không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp thanh toán không hợp lệ',
        );
      }

      return this.successResponse(
        'Thành công',
        this.mapToPaymentLinkResponse(parsed.data),
      );
    } catch (error) {
      this.logger.error('Lỗi tạo liên kết thanh toán:', error);
      return this.errorResponse('Tạo liên kết thanh toán thất bại');
    }
  }

  async getPaymentLinkInfo(
    orderId: string,
  ): Promise<ServiceResponse<PaymentLinkDto>> {
    try {
      const raw = await this.payOS.paymentRequests.get(orderId);
      const parsed = paymentLinkSchema.safeParse(raw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi getPaymentLinkInfo không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp thanh toán không hợp lệ',
        );
      }

      if (!parsed.data) {
        return this.errorResponse('Không tìm thấy đơn hàng');
      }

      return this.successResponse('Thành công', parsed.data);
    } catch (error) {
      this.logger.error('Lỗi lấy thông tin liên kết thanh toán:', error);
      return this.errorResponse('Lấy thông tin liên kết thanh toán thất bại');
    }
  }

  async cancelPaymentLink(
    orderId: string,
    cancellationReason: string,
  ): Promise<ServiceResponse<PaymentLinkDto | Record<string, unknown>>> {
    try {
      const raw = await this.payOS.paymentRequests.cancel(orderId, {
        cancellationReason,
      });
      const parsed = paymentLinkSchema.safeParse(raw);

      if (!parsed.success) {
        if (!raw) {
          return this.errorResponse('Hủy liên kết thanh toán thất bại');
        }
        return this.successResponse('Đã hủy liên kết thanh toán', raw);
      }

      return this.successResponse(
        'Hủy liên kết thanh toán thành công',
        parsed.data,
      );
    } catch (error) {
      this.logger.error('Lỗi hủy liên kết thanh toán:', error);
      return this.errorResponse('Hủy liên kết thanh toán thất bại');
    }
  }

  verifyPaymentWebhookData(
    webhookData: unknown,
  ): PaymentLinkDto | Record<string, unknown> {
    try {
      const raw = this.payOS.webhooks.verify(webhookData);
      const parsed = paymentLinkSchema.safeParse(raw);

      return parsed.success ? parsed.data : (raw as Record<string, unknown>);
    } catch (error) {
      this.logger.error('Lỗi xác minh webhook:', error);
      throw error;
    }
  }

  async confirmWebhook(webhookUrl: string): Promise<ServiceResponse<null>> {
    try {
      await this.payOS.webhooks.confirm(webhookUrl);
      return this.successResponse('Xác nhận webhook thành công', null);
    } catch (error) {
      this.logger.error('Lỗi xác nhận webhook:', error);
      return this.errorResponse('Xác nhận webhook thất bại');
    }
  }

  private mapToPaymentLinkResponse(
    paymentLink: PaymentLinkDto,
  ): PaymentLinkResponse {
    return {
      bin: paymentLink.bin ?? null,
      checkoutUrl: paymentLink.checkoutUrl ?? null,
      accountNumber: paymentLink.accountNumber ?? null,
      accountName: paymentLink.accountName ?? null,
      amount: paymentLink.amount ?? null,
      description: paymentLink.description ?? null,
      orderCode:
        typeof paymentLink.orderCode === 'number'
          ? paymentLink.orderCode
          : typeof paymentLink.orderCode === 'string'
            ? Number(paymentLink.orderCode)
            : null,
      qrCode: paymentLink.qrCode ?? null,
    };
  }

  private successResponse<T>(message: string, data: T): ServiceResponse<T> {
    return {
      error: 0,
      message,
      data,
    };
  }

  private errorResponse<T>(message: string): ServiceResponse<T> {
    return {
      error: -1,
      message,
      data: null,
    };
  }
}

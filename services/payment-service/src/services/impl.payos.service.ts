import { Injectable, Logger } from '@nestjs/common';
import { payOS } from '../libs/payos.config';
import {
  paymentLinkSchema,
  PaymentLinkDto,
  CreatePaymentLinkData,
  PaymentLinkResponse,
} from '../entities/payos.entity';
import { PayosService } from './payos.service';
import { transformKeys, toCamelCase, toSnakeCase } from '../libs/utils';
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
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      const camelData = transformKeys(data, toCamelCase);
      // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
      const raw = await this.payOS.paymentRequests.create(camelData);
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = paymentLinkSchema.safeParse(snakeRaw);

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
    order_id: string,
  ): Promise<ServiceResponse<PaymentLinkDto>> {
    try {
      const raw = await this.payOS.paymentRequests.get(order_id);
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = paymentLinkSchema.safeParse(snakeRaw);

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
    order_id: string,
    cancellation_reason: string,
  ): Promise<ServiceResponse<PaymentLinkDto | Record<string, unknown>>> {
    try {
      const raw = await this.payOS.paymentRequests.cancel(order_id, {
        cancellationReason: cancellation_reason,
      });
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = paymentLinkSchema.safeParse(snakeRaw);

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
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = paymentLinkSchema.safeParse(snakeRaw);

      return parsed.success
        ? parsed.data
        : (snakeRaw as Record<string, unknown>);
    } catch (error) {
      this.logger.error('Lỗi xác minh webhook:', error);
      throw error;
    }
  }

  async confirmWebhook(webhook_url: string): Promise<ServiceResponse<null>> {
    try {
      await this.payOS.webhooks.confirm(webhook_url);
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
      checkout_url: paymentLink.checkout_url ?? null,
      account_number: paymentLink.account_number ?? null,
      account_name: paymentLink.account_name ?? null,
      amount: paymentLink.amount ?? null,
      description: paymentLink.description ?? null,
      order_code:
        typeof paymentLink.order_code === 'number'
          ? paymentLink.order_code
          : typeof paymentLink.order_code === 'string'
            ? Number(paymentLink.order_code)
            : null,
      qr_code: paymentLink.qr_code ?? null,
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

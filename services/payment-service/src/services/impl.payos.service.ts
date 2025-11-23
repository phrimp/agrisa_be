import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { PayOS } from '@payos/node';
import {
  paymentLinkSchema,
  PaymentLinkDto,
  CreatePaymentLinkData,
  PaymentLinkResponse,
  payoutSchema,
  PayoutDto,
  CreatePayoutData,
  CreateBatchPayoutData,
  payoutAccountBalanceSchema,
  PayoutAccountBalanceDto,
  estimateCreditSchema,
  EstimateCreditDto,
} from '../types/payos.types';
import { PayosService } from './payos.service';
import { transformKeys, toCamelCase, toSnakeCase } from '../libs/utils';
import { payosConfig, validatePayosConfig } from '../libs/payos.config';
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
  payouts: {
    create: (data: CreatePayoutData) => Promise<unknown>;
    list: (options?: any) => Promise<unknown>;
    get: (payoutId: string) => Promise<unknown>;
    estimateCredit: (data: CreateBatchPayoutData) => Promise<unknown>;
  };
  batch: {
    create: (data: CreateBatchPayoutData) => Promise<unknown>;
  };
  payoutsAccount: {
    balance: () => Promise<unknown>;
  };
};

type ServiceResponse<T = unknown> = {
  error: number;
  message: string;
  data: T | null;
};

@Injectable()
export class ImplPayosService implements PayosService, OnModuleInit {
  private readonly logger = new Logger(ImplPayosService.name);
  private payOS: PayOSClient;

  constructor() {}

  onModuleInit() {
    validatePayosConfig();

    this.logger.log('PayOS Configuration loaded:', {
      hasClientId: !!payosConfig.clientId,
      hasApiKey: !!payosConfig.apiKey,
      hasChecksumKey: !!payosConfig.checksumKey,
      clientIdLength: payosConfig.clientId?.length,
      apiKeyLength: payosConfig.apiKey?.length,
      checksumKeyLength: payosConfig.checksumKey?.length,
    });

    this.payOS = new PayOS({
      clientId: payosConfig.clientId,
      apiKey: payosConfig.apiKey,
      checksumKey: payosConfig.checksumKey,
    }) as unknown as PayOSClient;
  }

  async createPaymentLink(
    data: CreatePaymentLinkData,
  ): Promise<ServiceResponse<PaymentLinkResponse>> {
    try {
      const payosData = {
        ...data,
        expired_at: Math.floor(data.expired_at.getTime() / 1000),
      };

      delete (payosData as any).type;

      const camelData = transformKeys(payosData, toCamelCase);

      this.logger.log('Sending to PayOS:', JSON.stringify(camelData, null, 2));

      const raw = await this.payOS.paymentRequests.create(camelData);
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
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = paymentLinkSchema.safeParse(snakeRaw);

      if (!parsed.success) {
        if (!raw) {
          return this.errorResponse('Hủy liên kết thanh toán thất bại');
        }
        return this.successResponse(
          'Đã hủy liên kết thanh toán',
          raw as Record<string, unknown>,
        );
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

  getExpiredDuration(): string {
    return payosConfig.expiredDuration;
  }

  getOrderCodeLength(): number {
    return payosConfig.orderCodeLength;
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

  // Payout methods
  async createPayout(
    data: CreatePayoutData,
  ): Promise<ServiceResponse<PayoutDto>> {
    try {
      const raw = await this.payOS.payouts.create(data);
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = payoutSchema.safeParse(snakeRaw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi createPayout không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp payout không hợp lệ',
        );
      }

      return this.successResponse('Tạo lệnh chi thành công', parsed.data);
    } catch (error) {
      this.logger.error('Lỗi tạo lệnh chi:', error);
      return this.errorResponse('Tạo lệnh chi thất bại');
    }
  }

  async createBatchPayout(
    data: CreateBatchPayoutData,
  ): Promise<ServiceResponse<PayoutDto>> {
    try {
      const raw = await this.payOS.batch.create(data);
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = payoutSchema.safeParse(snakeRaw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi createBatchPayout không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp payout không hợp lệ',
        );
      }

      return this.successResponse(
        'Tạo lệnh chi hàng loạt thành công',
        parsed.data,
      );
    } catch (error) {
      this.logger.error('Lỗi tạo lệnh chi hàng loạt:', error);
      return this.errorResponse('Tạo lệnh chi hàng loạt thất bại');
    }
  }

  async getPayout(payoutId: string): Promise<ServiceResponse<PayoutDto>> {
    try {
      const raw = await this.payOS.payouts.get(payoutId);
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = payoutSchema.safeParse(snakeRaw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi getPayout không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp payout không hợp lệ',
        );
      }

      return this.successResponse(
        'Lấy thông tin lệnh chi thành công',
        parsed.data,
      );
    } catch (error) {
      this.logger.error('Lỗi lấy thông tin lệnh chi:', error);
      return this.errorResponse('Lấy thông tin lệnh chi thất bại');
    }
  }

  async getPayouts(options?: {
    limit?: number;
    offset?: number;
    referenceId?: string;
    approvalState?: string;
    category?: string;
    fromDate?: string;
    toDate?: string;
  }): Promise<ServiceResponse<{ payouts: PayoutDto[]; pagination: any }>> {
    try {
      const raw = await this.payOS.payouts.list(options);
      const snakeRaw = transformKeys(raw, toSnakeCase);

      if (!snakeRaw || typeof snakeRaw !== 'object') {
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp payout không hợp lệ',
        );
      }

      const payouts = Array.isArray(snakeRaw.payouts)
        ? snakeRaw.payouts
            .map((p: any) => {
              const parsed = payoutSchema.safeParse(p);
              return parsed.success ? parsed.data : null;
            })
            .filter(Boolean)
        : [];

      return this.successResponse('Lấy danh sách lệnh chi thành công', {
        payouts,
        pagination: snakeRaw.pagination || {},
      });
    } catch (error) {
      this.logger.error('Lỗi lấy danh sách lệnh chi:', error);
      return this.errorResponse('Lấy danh sách lệnh chi thất bại');
    }
  }

  async estimatePayoutCredit(
    data: CreateBatchPayoutData,
  ): Promise<ServiceResponse<EstimateCreditDto>> {
    try {
      const raw = await this.payOS.payouts.estimateCredit(data);
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = estimateCreditSchema.safeParse(snakeRaw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi estimatePayoutCredit không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp payout không hợp lệ',
        );
      }

      return this.successResponse('Ước tính chi phí thành công', parsed.data);
    } catch (error) {
      this.logger.error('Lỗi ước tính chi phí:', error);
      return this.errorResponse('Ước tính chi phí thất bại');
    }
  }

  async getPayoutAccountBalance(): Promise<
    ServiceResponse<PayoutAccountBalanceDto>
  > {
    try {
      const raw = await this.payOS.payoutsAccount.balance();
      const snakeRaw = transformKeys(raw, toSnakeCase);
      const parsed = payoutAccountBalanceSchema.safeParse(snakeRaw);

      if (!parsed.success) {
        this.logger.error(
          'Phản hồi getPayoutAccountBalance không hợp lệ',
          parsed.error.format(),
        );
        return this.errorResponse(
          'Phản hồi từ nhà cung cấp payout không hợp lệ',
        );
      }

      return this.successResponse(
        'Lấy thông tin số dư thành công',
        parsed.data,
      );
    } catch (error) {
      this.logger.error('Lỗi lấy thông tin số dư:', error);
      return this.errorResponse('Lấy thông tin số dư thất bại');
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
      expired_at: paymentLink.expired_at ?? null,
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

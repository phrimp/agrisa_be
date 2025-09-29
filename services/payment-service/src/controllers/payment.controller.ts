import {
  Body,
  Controller,
  Get,
  Param,
  Post,
  Logger,
  Inject,
} from '@nestjs/common';
import {
  createPaymentLinkSchema,
  paymentLinkResponseSchema,
} from '../entities/payos.entity';
import type { CreatePaymentLinkData } from '../entities/payos.entity';
import type { PayosService } from '../services/payos.service';
import type { PaymentService } from '../services/payment.service';

@Controller('payment')
export class PaymentController {
  private readonly logger = new Logger(PaymentController.name);

  constructor(
    @Inject('PayosService') private readonly payosService: PayosService,
    @Inject('PaymentService') private readonly paymentService: PaymentService,
  ) {}

  @Post('link')
  async createPaymentLink(
    @Body() body: CreatePaymentLinkData & { userId: string },
  ) {
    const { userId, ...payosData } = body;

    const parsed = createPaymentLinkSchema.safeParse(payosData);
    if (!parsed.success) {
      this.logger.warn(
        'Invalid createPaymentLink payload',
        parsed.error.format(),
      );
      return {
        error: -1,
        message: 'Dữ liệu không hợp lệ',
        data: null,
      };
    }

    try {
      await this.paymentService.create({
        id: parsed.data.orderCode.toString(),
        amount: parsed.data.amount,
        description: parsed.data.description,
        userId: userId,
        status: 'pending',
      });

      return this.payosService.createPaymentLink(parsed.data);
    } catch (error) {
      this.logger.error('Failed to create payment', error);
      return {
        error: -1,
        message: 'Tạo thanh toán thất bại',
        data: null,
      };
    }
  }

  @Get('link/:orderId')
  async getPaymentLinkInfo(@Param('orderId') orderId: string) {
    return this.payosService.getPaymentLinkInfo(orderId);
  }

  @Post('link/:orderId/cancel')
  async cancelPaymentLink(
    @Param('orderId') orderId: string,
    @Body('cancellationReason') cancellationReason: string,
  ) {
    if (!cancellationReason) {
      return {
        error: -1,
        message: 'Vui lòng cung cấp cancellationReason',
        data: null,
      };
    }

    return this.payosService.cancelPaymentLink(orderId, cancellationReason);
  }

  @Post('webhook/verify')
  verifyWebhook(@Body() body: unknown) {
    try {
      const raw = this.payosService.verifyPaymentWebhookData(body);

      const parsed = paymentLinkResponseSchema.safeParse(raw);
      if (parsed.success) {
        return {
          error: 0,
          message: 'Thành công',
          data: parsed.data,
        };
      }

      return {
        error: 0,
        message: 'Thành công',
        data: raw as Record<string, unknown>,
      };
    } catch (err) {
      this.logger.error('Failed to verify webhook', err);
      return {
        error: -1,
        message: 'Xác minh webhook thất bại',
        data: null,
      };
    }
  }

  @Post('webhook/confirm')
  async confirmWebhook(@Body('webhookUrl') webhookUrl: string) {
    if (!webhookUrl) {
      return {
        error: -1,
        message: 'webhookUrl is required',
        data: null,
      };
    }

    return this.payosService.confirmWebhook(webhookUrl);
  }
}

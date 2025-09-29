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
import { generateRandomString } from 'src/libs/utils';
import { PAYOS_EXPIRED_DURATION } from 'src/libs/payos.config';

const ORDER_CODE_LENGTH = 6;

@Controller('payment')
export class PaymentController {
  private readonly logger = new Logger(PaymentController.name);

  constructor(
    @Inject('PayosService') private readonly payosService: PayosService,
    @Inject('PaymentService') private readonly paymentService: PaymentService,
  ) {}

  @Post('link')
  async createPaymentLink(
    @Body() body: CreatePaymentLinkData & { user_id: string },
  ) {
    const { user_id, ...payosData } = body;

    const cleanedPayosData = Object.fromEntries(
      Object.entries(payosData).filter(([, value]) => value !== undefined),
    );

    const parsed = createPaymentLinkSchema.safeParse(cleanedPayosData);
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
      const orderCode =
        parsed.data.order_code ??
        Math.floor(Math.random() * 10 ** ORDER_CODE_LENGTH);

      const durationStr = PAYOS_EXPIRED_DURATION || '';
      let durationSeconds: number;
      if (durationStr.includes('*')) {
        const parts = durationStr.split('*').map((s) => parseInt(s.trim()));
        durationSeconds = parts.reduce((a, b) => a * b, 1);
      } else {
        durationSeconds = parseInt(durationStr);
      }
      const expiredAt = new Date(Date.now() + durationSeconds * 1000);

      const paymentId = generateRandomString();

      await this.paymentService.create({
        id: paymentId,
        order_code: orderCode.toString(),
        amount: parsed.data.amount,
        description: parsed.data.description,
        user_id: user_id,
        expired_at: expiredAt,
      });

      const payosPayload = {
        ...parsed.data,
        order_code: orderCode,
        return_url: parsed.data.return_url,
        cancel_url: parsed.data.cancel_url,
        expired_at: expiredAt,
      };

      const payosResponse =
        await this.payosService.createPaymentLink(payosPayload);

      if (payosResponse.error === 0 && payosResponse.data?.checkout_url) {
        await this.paymentService.update(paymentId, {
          checkout_url: payosResponse.data.checkout_url,
        });
      }

      return payosResponse;
    } catch (error) {
      this.logger.error('Failed to create payment', error);
      return {
        error: -1,
        message: 'Tạo thanh toán thất bại',
        data: null,
      };
    }
  }

  @Get('link/:order_id')
  async getPaymentLinkInfo(@Param('order_id') order_id: string) {
    return this.payosService.getPaymentLinkInfo(order_id);
  }

  @Post('link/:order_id/cancel')
  async cancelPaymentLink(
    @Param('order_id') order_id: string,
    @Body('cancellation_reason') cancellation_reason: string,
  ) {
    if (!cancellation_reason) {
      return {
        error: -1,
        message: 'Vui lòng cung cấp cancellation_reason',
        data: null,
      };
    }

    return this.payosService.cancelPaymentLink(order_id, cancellation_reason);
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
  async confirmWebhook(@Body('webhook_url') webhook_url: string) {
    if (!webhook_url) {
      return {
        error: -1,
        message: 'webhook_url is required',
        data: null,
      };
    }

    return this.payosService.confirmWebhook(webhook_url);
  }
}

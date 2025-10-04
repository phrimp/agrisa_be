import {
  Body,
  Controller,
  Get,
  Param,
  Post,
  Logger,
  Inject,
  Headers,
  Delete,
  Query,
  HttpStatus,
} from '@nestjs/common';
import {
  createPaymentLinkSchema,
  paymentLinkResponseSchema,
} from '../types/payos.types';
import type { CreatePaymentLinkData } from '../types/payos.types';
import type { PayosService } from '../services/payos.service';
import type { PaymentService } from '../services/payment.service';
import { checkPermissions, generateRandomString } from 'src/libs/utils';
import { PAYOS_EXPIRED_DURATION } from 'src/libs/payos.config';

const ORDER_CODE_LENGTH = 6;

@Controller()
export class PaymentController {
  private readonly logger = new Logger(PaymentController.name);

  constructor(
    @Inject('PayosService') private readonly payosService: PayosService,
    @Inject('PaymentService') private readonly paymentService: PaymentService,
  ) {}

  @Post('protected/link')
  async createPaymentLink(
    @Body() body: CreatePaymentLinkData,
    @Headers('x-user-id') user_id: string,
  ) {
    const payosData = body;

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

  @Get('protected/link/:order_id')
  async getPaymentLinkInfo(@Param('order_id') order_id: string) {
    return this.payosService.getPaymentLinkInfo(order_id);
  }

  @Delete('protected/link/:order_id')
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

  @Post('public/webhook/verify')
  async verifyWebhook(@Body() body: unknown) {
    try {
      const raw = this.payosService.verifyPaymentWebhookData(body);

      const parsed = paymentLinkResponseSchema.safeParse(raw);
      if (parsed.success) {
        if (parsed.data.order_code) {
          const payment = await this.paymentService.findById(
            parsed.data.order_code.toString(),
          );
          if (payment && parsed.data.qr_code) {
            await this.paymentService.update(payment.id, {
              status: 'completed',
            });
          }
        }

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

  @Post('public/webhook/confirm')
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

  @Get('protected/orders')
  async getAllOrders(
    @Headers('x-user-id') user_id: string,
    @Headers('x-user-permissions') user_permissions: string,
    @Query('page') page = '1',
    @Query('limit') limit = '10',
    @Query('status') status?: string[],
  ) {
    const page_num = Math.max(parseInt(page, 10) || 1, 1);
    const limit_num = Math.max(parseInt(limit, 10) || 10, 1);
    const permissions = user_permissions ? user_permissions.split(',') : [];

    try {
      if (checkPermissions(permissions, ['admin'])) {
        const orders = await this.paymentService.find(
          page_num,
          limit_num,
          status,
        );
        return {
          message: 'Thành công',
          code: HttpStatus.OK,
          data: orders,
          total_pages: Math.ceil(orders.length / limit_num),
          current_page: page_num,
          total_items: orders.length,
          items_per_page: limit_num,
          previous: page_num > 1,
          next: page_num * limit_num < orders.length,
        };
      } else {
        const orders = await this.paymentService.findByUserId(
          user_id,
          page_num,
          limit_num,
          status,
        );
        return {
          message: 'Thành công',
          code: HttpStatus.OK,
          data: orders,
          total_pages: Math.ceil(orders.length / limit_num),
          current_page: page_num,
          total_items: orders.length,
          items_per_page: limit_num,
          previous: page_num > 1,
          next: page_num * limit_num < orders.length,
        };
      }
    } catch (error) {
      return {
        message: `Lỗi: ${error || 'Đã xảy ra lỗi'}`,
        code: HttpStatus.INTERNAL_SERVER_ERROR,
        data: null,
        total_pages: 0,
        current_page: page_num,
        total_items: 0,
        items_per_page: limit_num,
        previous: false,
        next: false,
      };
    }
  }
}

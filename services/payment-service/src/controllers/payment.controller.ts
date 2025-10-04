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
  HttpException,
  HttpStatus,
} from '@nestjs/common';
import {
  createPaymentLinkSchema,
  webhookPayloadSchema,
} from '../types/payos.types';
import type { CreatePaymentLinkData } from '../types/payos.types';
import type { PayosService } from '../services/payos.service';
import type { PaymentService } from '../services/payment.service';
import { checkPermissions, generateRandomString } from 'src/libs/utils';
import { PAYOS_EXPIRED_DURATION } from 'src/libs/payos.config';
import { paymentViewSchema } from 'src/types/payment.types';
import z from 'zod';
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
      throw new HttpException('Dữ liệu không hợp lệ', HttpStatus.BAD_REQUEST);
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
      throw new HttpException(
        'Tạo thanh toán thất bại',
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
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
      throw new HttpException(
        'Vui lòng cung cấp cancellation_reason',
        HttpStatus.BAD_REQUEST,
      );
    }

    return this.payosService.cancelPaymentLink(order_id, cancellation_reason);
  }

  @Post('public/webhook/verify')
  async verifyWebhook(@Body() body: unknown) {
    this.logger.log(`Webhook body received: ${JSON.stringify(body)}`); // Thêm log để debug
    try {
      const raw = this.payosService.verifyPaymentWebhookData(body);
      this.logger.log(`Raw webhook data: ${JSON.stringify(raw)}`); // Thêm log

      const parsed = webhookPayloadSchema.safeParse(raw); // Dùng schema mới
      this.logger.log(`Parsed webhook data: ${JSON.stringify(parsed)}`); // Thêm log

      if (parsed.success) {
        if (parsed.data.data && parsed.data.data.order_code) {
          // Sửa từ orderCode thành order_code (snake_case)
          const payment = await this.paymentService.findById(
            parsed.data.data.order_code.toString(), // Sửa từ orderCode thành order_code
          );
          if (payment) {
            // Check data.code === '00' cho thanh toán thành công
            if (String(parsed.data.code) === '00') {
              await this.paymentService.update(payment.id, {
                status: 'completed',
              });
              this.logger.log(`Payment ${payment.id} updated to completed`);
            } else {
              this.logger.warn(
                `Webhook received but not successful: code=${parsed.data.code}`,
              );
            }
          } else {
            this.logger.warn(
              `Payment not found for order_code: ${parsed.data.data.order_code}`, // Sửa từ orderCode thành order_code
            );
          }
        } else {
          this.logger.warn('No order_code in webhook data'); // Sửa từ orderCode thành order_code
        }

        return parsed.data;
      }

      throw new HttpException(
        'Dữ liệu webhook không hợp lệ',
        HttpStatus.BAD_REQUEST,
      );
    } catch (err) {
      this.logger.error('Failed to verify webhook', err);
      throw new HttpException(
        'Xác minh webhook thất bại',
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
    }
  }

  @Post('public/webhook/confirm')
  async confirmWebhook(@Body('webhook_url') webhook_url: string) {
    if (!webhook_url) {
      throw new HttpException(
        'webhook_url is required',
        HttpStatus.BAD_REQUEST,
      );
    }

    return this.payosService.confirmWebhook(webhook_url);
  }

  @Get('protected/orders')
  getAllOrders(
    @Headers('x-user-id') user_id: string,
    @Headers('x-user-permissions') user_permissions: string,
    @Query('page') page = '1',
    @Query('limit') limit = '10',
    @Query('status') status?: string,
  ) {
    const page_num = Math.max(parseInt(page, 10) || 1, 1);
    const limit_num = Math.max(parseInt(limit, 10) || 10, 1);
    const permissions = user_permissions ? user_permissions.split(',') : [];

    const orders = checkPermissions(permissions, ['view_all_orders'])
      ? this.paymentService.find(page_num, limit_num, status?.split(','))
      : this.paymentService.findByUserId(
          user_id,
          page_num,
          limit_num,
          status?.split(','),
        );

    return orders
      .then((res) => {
        const { items, total } = res;
        const total_pages = Math.ceil(total / limit_num);
        return {
          items: z.array(paymentViewSchema).parse(items),
          metadata: {
            page: page_num,
            limit: limit_num,
            total_items: total,
            total_pages,
            next: page_num < total_pages,
            previous: page_num > 1,
          },
        };
      })
      .catch((err) => {
        this.logger.error('Failed to get orders', err);
        throw new HttpException(
          'Lỗi khi lấy danh sách đơn hàng',
          HttpStatus.INTERNAL_SERVER_ERROR,
        );
      });
  }
}

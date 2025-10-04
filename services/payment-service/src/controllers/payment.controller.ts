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
const ORDER_CODE_LENGTH = 8;

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
    const payos_data = body;

    const cleaned_payos_data = Object.fromEntries(
      Object.entries(payos_data).filter(([, value]) => value !== undefined),
    );

    const parsed = createPaymentLinkSchema.safeParse(cleaned_payos_data);
    if (!parsed.success) {
      this.logger.warn(
        'Invalid createPaymentLink payload',
        parsed.error.format(),
      );
      throw new HttpException('Dữ liệu không hợp lệ', HttpStatus.BAD_REQUEST);
    }

    try {
      const order_code =
        parsed.data.order_code ??
        Math.floor(Math.random() * 10 ** ORDER_CODE_LENGTH);

      const duration_str = PAYOS_EXPIRED_DURATION || '';
      let duration_seconds: number;
      if (duration_str.includes('*')) {
        const parts = duration_str.split('*').map((s) => parseInt(s.trim()));
        duration_seconds = parts.reduce((a, b) => a * b, 1);
      } else {
        duration_seconds = parseInt(duration_str);
      }
      const expired_at = new Date(Date.now() + duration_seconds * 1000);

      const payment_id = generateRandomString();

      await this.paymentService.create({
        id: payment_id,
        order_code: order_code.toString(),
        amount: parsed.data.amount,
        description: parsed.data.description,
        user_id: user_id,
        expired_at: expired_at,
      });

      const payos_payload = {
        ...parsed.data,
        order_code: order_code,
        return_url: parsed.data.return_url,
        cancel_url: parsed.data.cancel_url,
        expired_at: expired_at,
      };

      const payos_response =
        await this.payosService.createPaymentLink(payos_payload);

      if (payos_response.error === 0 && payos_response.data?.checkout_url) {
        await this.paymentService.update(payment_id, {
          checkout_url: payos_response.data.checkout_url,
        });
      }

      return payos_response.data;
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
    const payment_link_info =
      await this.payosService.getPaymentLinkInfo(order_id);
    return payment_link_info.data;
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

    return this.payosService
      .cancelPaymentLink(order_id, cancellation_reason)
      .then((cancel_response) => cancel_response.data);
  }

  @Post('public/webhook/verify')
  async verifyWebhook(@Body() body: unknown) {
    try {
      this.logger.log('Received webhook payload', { body }); // Thêm log cho payload nhận được

      this.payosService.verifyPaymentWebhookData(body);

      const parsed = webhookPayloadSchema.safeParse(body);
      if (parsed.success) {
        this.logger.log('Webhook parsed successfully', {
          parsedData: parsed.data,
        }); // Thêm log cho dữ liệu đã parse

        if (parsed.data.data && parsed.data.data.orderCode) {
          const payment = await this.paymentService.findByOrderCode(
            parsed.data.data.orderCode.toString(),
          );
          if (payment) {
            if (parsed.data.data.code === '00') {
              await this.paymentService.update(payment.id, {
                status: 'completed',
                paid_at: new Date(),
              });
              this.logger.log('Payment status updated to completed', {
                orderCode: parsed.data.data.orderCode,
              }); // Thêm log cho cập nhật trạng thái
            } else {
              this.logger.log('Payment code not 00, no status update', {
                code: parsed.data.data.code,
                orderCode: parsed.data.data.orderCode,
              }); // Thêm log cho trường hợp không cập nhật
            }
          } else {
            this.logger.warn('Payment not found for orderCode', {
              orderCode: parsed.data.data.orderCode,
            }); // Thêm log cảnh báo nếu không tìm thấy payment
          }
        }

        return parsed.data;
      }

      this.logger.warn('Webhook payload validation failed', {
        errors: parsed.error.format(),
      }); // Thêm log cho lỗi validation
      throw new HttpException(
        'Dữ liệu webhook không hợp lệ',
        HttpStatus.BAD_REQUEST,
      );
    } catch (error) {
      this.logger.error('Failed to verify webhook', error);
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

    return this.payosService
      .confirmWebhook(webhook_url)
      .then((confirm_response) => confirm_response.data);
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

    const orders_result = checkPermissions(permissions, ['view_all_orders'])
      ? this.paymentService.find(page_num, limit_num, status?.split(','))
      : this.paymentService.findByUserId(
          user_id,
          page_num,
          limit_num,
          status?.split(','),
        );

    return orders_result
      .then((result) => {
        const { items, total } = result;
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
      .catch((error) => {
        this.logger.error('Failed to get orders', error);
        throw new HttpException(
          'Lỗi khi lấy danh sách đơn hàng',
          HttpStatus.INTERNAL_SERVER_ERROR,
        );
      });
  }
}

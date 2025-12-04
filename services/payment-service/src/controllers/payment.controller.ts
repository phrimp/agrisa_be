import {
  Body,
  Controller,
  Delete,
  Get,
  Headers,
  HttpException,
  HttpStatus,
  Inject,
  Logger,
  Param,
  Post,
  Query,
  Res,
} from '@nestjs/common';
import type { Response } from 'express';
import { publisher } from 'src/events/publisher';
import { payosConfig } from 'src/libs/payos.config';
import { checkPermissions, generateRandomString } from 'src/libs/utils';
import type { OrderItemService } from 'src/services/order-item.service';
import { paymentViewSchema } from 'src/types/payment.types';
import { payoutViewSchema } from 'src/types/payout.types';
import z from 'zod';
import type { PaymentService } from '../services/payment.service';
import type { PayosService } from '../services/payos.service';
import type { PayoutService } from '../services/payout.service';
import type { CreatePaymentLinkData } from '../types/payos.types';
import {
  createPaymentLinkSchema,
  webhookPayloadSchema,
} from '../types/payos.types';

@Controller()
export class PaymentController {
  private readonly logger = new Logger(PaymentController.name);

  constructor(
    @Inject('PayosService') private readonly payosService: PayosService,
    @Inject('PaymentService') private readonly paymentService: PaymentService,
    @Inject('OrderItemService')
    private readonly orderItemService: OrderItemService,
    @Inject('PayoutService') private readonly payoutService: PayoutService,
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
        Math.floor(Math.random() * 10 ** payosConfig.orderCodeLength);

      const duration_str = this.payosService.getExpiredDuration();
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
        type: payos_data.type,
        user_id: user_id,
        expired_at: expired_at,
      });

      if (parsed.data.items && parsed.data.items.length > 0) {
        for (const item of parsed.data.items) {
          await this.orderItemService.create({
            id: generateRandomString(),
            payment_id: payment_id,
            item_id: item.item_id,
            name: item.name,
            price: item.price,
            quantity: item.quantity ?? 1,
            created_at: new Date(),
            updated_at: new Date(),
          });
        }
      }

      // Remove item_id from items before sending to PayOS
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const items = parsed.data.items?.map(({ item_id, ...item }) => item);

      const payos_payload = {
        ...parsed.data,
        items: items,
        order_code: order_code,
        return_url: parsed.data.return_url,
        cancel_url: `https://agrisa-api.phrimp.io.vn/payment/public/webhook/cancel?order_id=${order_code}&redirect=${encodeURIComponent(parsed.data.cancel_url!)}`,
        expired_at: expired_at,
      };

      const payos_response =
        await this.payosService.createPaymentLink(payos_payload);

      if (payos_response.error !== 0) {
        this.logger.error('PayOS createPaymentLink failed', payos_response);
        throw new HttpException(
          payos_response.message || 'Tạo liên kết thanh toán thất bại',
          HttpStatus.BAD_REQUEST,
        );
      }

      if (payos_response.data?.checkout_url) {
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
    @Body('cancellation_reason') cancellation_reason?: string,
  ) {
    return this.payosService
      .cancelPaymentLink(order_id, cancellation_reason || '')
      .then((cancel_response) => cancel_response.data);
  }

  @Get('public/webhook/cancel')
  async handleCancelRedirect(
    @Query('order_id') order_id: string,
    @Query('redirect') redirect_url: string,
    @Res() res: Response,
  ) {
    if (!order_id || !redirect_url) {
      throw new HttpException(
        'order_id and redirect are required',
        HttpStatus.BAD_REQUEST,
      );
    }

    try {
      await this.payosService.cancelPaymentLink(order_id, 'Khách hàng hủy');
      const payment = await this.paymentService.findByOrderCode(order_id);
      if (payment) {
        await this.paymentService.update(payment.id, { status: 'cancelled' });
      }
      return res.redirect(redirect_url);
    } catch (error) {
      this.logger.error('Failed to cancel payment link', error);
      return res.redirect(`${redirect_url}?error=cancel_failed`);
    }
  }

  @Post('public/webhook/verify')
  async verifyWebhook(@Body() body: unknown) {
    try {
      this.payosService.verifyPaymentWebhookData(body);

      const parsed = webhookPayloadSchema.safeParse(body);
      if (parsed.success) {
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
              const publisher_payment =
                await this.paymentService.findByOrderCode(
                  parsed.data.data.orderCode.toString(),
                );
              if (publisher_payment) {
                this.logger.log('Publishing payment to queue', {
                  orderCode: parsed.data.data.orderCode,
                  type: publisher_payment.type,
                  amount: publisher_payment.amount,
                  orderItemsCount: publisher_payment.orderItems?.length || 0,
                });
                await publisher(publisher_payment);
                this.logger.log('Payment event published to queue', {
                  orderCode: parsed.data.data.orderCode,
                });
              } else {
                this.logger.error(
                  'Failed to fetch updated payment for publishing',
                  {
                    orderCode: parsed.data.data.orderCode,
                  },
                );
              }
              console.log('DATA:', parsed.data.data);
            }
          } else {
            this.logger.warn('Không tìm thấy order_code', {
              orderCode: parsed.data.data.orderCode,
            });
          }
        }

        return parsed.data;
      }

      this.logger.warn('Webhook payload validation failed', {
        errors: parsed.error,
      });
      throw new HttpException(
        'Dữ liệu webhook không hợp lệ',
        HttpStatus.BAD_REQUEST,
      );
    } catch (error) {
      this.logger.error('Thất bại xác minh webhook', error);
      throw new HttpException(
        'Xác minh webhook thất bại',
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
    }
  }

  @Post('public/webhook/confirm')
  async confirmWebhook(@Body('webhook_url') webhook_url: string) {
    if (!webhook_url) {
      throw new HttpException('yêu cầu webhook_url', HttpStatus.BAD_REQUEST);
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

  @Get('protected/order/:id')
  getOrderById(@Headers('x-user-id') user_id: string, @Param('id') id: string) {
    return this.paymentService.findByIdAndUserId(id, user_id);
  }

  @Post('public/payout')
  async createPayout(
    @Body()
    body: {
      amount: number;
      bank_code: string;
      account_number: string;
      user_id: string;
      description?: string;
    },
  ) {
    const { amount, bank_code, account_number, user_id, description } = body;

    const payout_id = generateRandomString();

    await this.payoutService.create({
      id: payout_id,
      amount,
      description: description || 'Chi trả bảo hiểm',
      user_id,
      bank_code,
      account_number,
      status: 'pending',
    });

    const url = new URL(
      `https://img.vietqr.io/image/${bank_code}-${account_number}-compact2.png`,
    );

    url.searchParams.set('amount', amount.toString());
    url.searchParams.set('addInfo', 'ChiTraBaoHiem');

    return {
      success: true,
      data: {
        payout_id,
        qr: url.toString(),
        verify_hook: `https://agrisa-api.phrimp.io.vn/payment/public/payout/verify?payout_id=${payout_id}`,
      },
    };
  }

  @Get('public/payout/verify')
  async verifyPayout(@Query('payout_id') payout_id: string) {
    if (!payout_id) {
      throw new HttpException('payout_id is required', HttpStatus.BAD_REQUEST);
    }

    const payout = await this.payoutService.findById(payout_id);
    if (!payout) {
      throw new HttpException('Payout not found', HttpStatus.NOT_FOUND);
    }

    await this.payoutService.update(payout_id, {
      status: 'completed',
      completed_at: new Date(),
    });

    return { success: true };
  }

  @Get('protected/payout')
  getPayout(
    @Headers('x-user-id') user_id: string,
    @Query('page') page = '1',
    @Query('limit') limit = '10',
  ) {
    const page_num = Math.max(parseInt(page, 10) || 1, 1);
    const limit_num = Math.max(parseInt(limit, 10) || 10, 1);

    return this.payoutService
      .findByUserId(user_id, page_num, limit_num)
      .then((result) => {
        const { items, total } = result;
        const total_pages = Math.ceil(total / limit_num);
        return {
          items: z.array(payoutViewSchema).parse(items),
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
        this.logger.error('Failed to get payouts', error);
        throw new HttpException(
          'Lỗi khi lấy danh sách chi trả',
          HttpStatus.INTERNAL_SERVER_ERROR,
        );
      });
  }

  @Get('protected/payout/:id')
  getPayoutById(
    @Headers('x-user-id') user_id: string,
    @Param('id') id: string,
  ) {
    return this.payoutService.findByIdAndUserId(id, user_id);
  }
}

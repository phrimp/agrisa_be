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
import type { ItemService } from 'src/services/item.service';
import { paymentViewSchema } from 'src/types/payment.types';
import { payoutViewSchema } from 'src/types/payout.types';
import z from 'zod';
import type { PaymentService } from '../services/payment.service';
import type { PayosService } from '../services/payos.service';
import type { PayoutService } from '../services/payout.service';
import type {
  CreatePaymentLinkData,
  CreatePayoutData,
} from '../types/payos.types';
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
    @Inject('ItemService')
    private readonly itemService: ItemService,
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
          await this.itemService.create({
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
                  orderItemsCount: publisher_payment.items?.length || 0,
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
  async getAllOrders(
    @Headers('x-user-id') user_id: string,
    @Headers('x-user-permissions') user_permissions: string,
    @Query('page') page = '1',
    @Query('limit') limit = '10',
    @Query('status') status?: string,
  ) {
    const page_num = Math.max(parseInt(page, 10) || 1, 1);
    const limit_num = Math.max(parseInt(limit, 10) || 10, 1);
    const permissions = user_permissions ? user_permissions.split(',') : [];

    try {
      // Lấy payments
      const payments_result = checkPermissions(permissions, ['view_all_orders'])
        ? await this.paymentService.find(
            page_num,
            limit_num,
            status?.split(','),
          )
        : await this.paymentService.findByUserId(
            user_id,
            page_num,
            limit_num,
            status?.split(','),
          );

      // Lấy payouts của user
      const payouts_result = await this.payoutService.findByUserId(
        user_id,
        page_num,
        limit_num,
      );

      const total = payments_result.total + payouts_result.total;
      const total_pages = Math.ceil(total / limit_num);

      return {
        payments: z.array(paymentViewSchema).parse(payments_result.items),
        payouts: z.array(payoutViewSchema).parse(payouts_result.items),
        metadata: {
          page: page_num,
          limit: limit_num,
          total_items: total,
          total_pages,
          next: page_num < total_pages,
          previous: page_num > 1,
        },
      };
    } catch (error) {
      this.logger.error('Failed to get orders', error);
      throw new HttpException(
        'Lỗi khi lấy danh sách đơn hàng',
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
    }
  }

  @Get('protected/order/:id')
  getOrderById(@Headers('x-user-id') user_id: string, @Param('id') id: string) {
    return this.paymentService.findByIdAndUserId(id, user_id);
  }

  @Post('protected/payout')
  async createPayout(
    @Headers('x-user-id') created_by: string,
    @Body()
    body: CreatePayoutData,
  ) {
    const {
      amount,
      bank_code,
      account_number,
      user_id,
      description,
      items,
      type,
    } = body;

    const payout_id = generateRandomString();

    const payment_id = generateRandomString();

    await this.paymentService.create({
      id: payment_id,
      amount,
      description: description || 'Chi trả bảo hiểm',
      status: 'pending',
      user_id: created_by,
      type,
    });

    if (items && items.length > 0) {
      for (const item of items) {
        await this.itemService.create({
          id: generateRandomString(),
          payment_id: payment_id,
          item_id: item.item_id,
          name: item.name,
          price: item.price,
          quantity: item.quantity ?? 1,
          payout_id,
          created_at: new Date(),
          updated_at: new Date(),
        });
      }
    }

    await this.payoutService.create({
      id: payout_id,
      amount,
      description: description || 'Chi trả bảo hiểm',
      user_id,
      bank_code,
      account_number,
      status: 'pending',
      payment_id,
    });

    // Update items with payout_id
    if (items && items.length > 0) {
      for (const item of items) {
        // Find the created item by item_id and payment_id, then update with payout_id
        const createdItem = await this.itemService.findByItemId(item.item_id!);
        if (createdItem) {
          await this.itemService.update(createdItem.id, { payout_id });
        }
      }
    }

    const url = new URL(
      `https://img.vietqr.io/image/${bank_code}-${account_number}-compact2.png`,
    );

    url.searchParams.set('amount', amount.toString());
    url.searchParams.set('addInfo', 'ChiTraBaoHiem');

    return {
      payout_id,
      qr: url.toString(),
      verify_hook: `https://agrisa-api.phrimp.io.vn/payment/public/payout/scan?payout_id=${payout_id}`,
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

    if (payout.payment_id) {
      await this.paymentService.update(payout.payment_id, {
        status: 'completed',
        paid_at: new Date(),
      });

      const publisher_payment = await this.paymentService.findById(
        payout.payment_id,
      );
      if (publisher_payment) {
        this.logger.log('Publishing payout payment to queue', {
          payment_id: payout.payment_id,
          type: publisher_payment.type,
          amount: publisher_payment.amount,
        });
        await publisher(publisher_payment);
        this.logger.log('Payout payment event published to queue', {
          payment_id: payout.payment_id,
        });
      } else {
        this.logger.error('Failed to fetch created payment for publishing', {
          payment_id: payout.payment_id,
        });
      }
    }

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

  @Get('protected/total')
  getTotalPayments(
    @Headers('x-user-id') user_id: string,
    @Query('type') type: string,
  ) {
    return this.paymentService.getTotalAmountByUserAndType(user_id, type);
  }

  @Get('protected/total/admin')
  getTotalPaymentsAdmin(@Query('type') type: string) {
    return this.paymentService.getTotalAmountByType(type);
  }

  // @Get('protected/payout/total')
  // getTotalPayouts(
  //   @Headers('x-user-id') user_id: string,
  //   @Query('type') type: string,
  // ) {
  //   return this.payoutService.getTotalPayoutAmountByTypeAndUserId(
  //     type,
  //     user_id,
  //   );
  // }

  @Get('public/payout/scan')
  async scanPayouts(@Query('payout_id') payout_id?: string) {
    if (!payout_id) {
      throw new HttpException('payout_id is required', HttpStatus.BAD_REQUEST);
    }

    const payout = await this.payoutService.findById(payout_id);
    if (!payout) {
      throw new HttpException('Payout not found', HttpStatus.NOT_FOUND);
    }

    await this.payoutService.update(payout_id, {
      status: 'scanned',
      completed_at: new Date(),
    });

    if (payout.payment_id) {
      await this.paymentService.update(payout.payment_id, {
        status: 'pending',
        paid_at: new Date(),
      });
    }
  }

  @Post('public/payout/verify/bulk')
  async bulkVerifyPayouts(@Body('item_ids') item_ids: string[]) {
    if (!item_ids || !Array.isArray(item_ids) || item_ids.length === 0) {
      throw new HttpException('item_ids is required', HttpStatus.BAD_REQUEST);
    }

    // Get items by item_ids to find payout_ids
    const items = await Promise.all(
      item_ids.map((item_id) => this.itemService.findByItemId(item_id)),
    );

    this.logger.log(
      'Found items:',
      items.map((item) => ({
        id: item?.id,
        item_id: item?.item_id,
        payout_id: item?.payout_id,
        payment_id: item?.payment_id,
      })),
    );

    const validItems = items.filter((item) => item !== null);

    this.logger.log('Valid items count:', validItems.length);

    // Collect payout_ids from items that have payout_id, or find via payment for others
    const payout_ids: string[] = [];
    for (const item of validItems) {
      if (item.payout_id) {
        payout_ids.push(item.payout_id);
        this.logger.log(`Item ${item.id} has payout_id: ${item.payout_id}`);
      } else {
        // Try to find payout via payment_id for legacy data
        this.logger.log(
          `Item ${item.id} has no payout_id, trying to find via payment ${item.payment_id}`,
        );
        // For now, skip legacy items - they need manual database update
        this.logger.warn(
          `Skipping legacy item ${item.id} - run SQL update to fix: UPDATE items SET payout_id = payouts.id FROM payouts WHERE items.payment_id = payouts.payment_id AND items.payout_id IS NULL;`,
        );
      }
    }

    this.logger.log('Final payout_ids:', payout_ids);

    if (payout_ids.length === 0) {
      throw new HttpException(
        'No valid payouts found for the provided item_ids',
        HttpStatus.BAD_REQUEST,
      );
    }

    // Process unique payouts to avoid duplicate operations
    const uniquePayoutIds = [...new Set(payout_ids)];
    const processedPayouts: Set<string> = new Set();

    for (const payout_id of uniquePayoutIds) {
      try {
        const payout = await this.payoutService.findById(payout_id);
        if (!payout) {
          this.logger.error(`Payout ${payout_id} not found`);
          continue;
        }

        if (payout.status === 'scanned') {
          await this.payoutService.update(payout_id, {
            status: 'completed',
            completed_at: new Date(),
          });

          if (payout.payment_id) {
            await this.paymentService.update(payout.payment_id, {
              status: 'completed',
              paid_at: new Date(),
            });

            const publisher_payment = await this.paymentService.findById(
              payout.payment_id,
            );
            if (publisher_payment) {
              this.logger.log('Publishing payout payment to queue', {
                payment_id: payout.payment_id,
                type: publisher_payment.type,
                amount: publisher_payment.amount,
              });
              await publisher(publisher_payment);
              this.logger.log('Payout payment event published to queue', {
                payment_id: payout.payment_id,
              });
            } else {
              this.logger.error(
                'Failed to fetch created payment for publishing',
                {
                  payment_id: payout.payment_id,
                },
              );
            }
          }

          processedPayouts.add(payout_id);
        } else {
          this.logger.warn(
            `Payout ${payout_id} status is ${payout.status}, not scanned`,
          );
        }
      } catch (error) {
        this.logger.error(`Failed to process payout ${payout_id}`, error);
      }
    }

    const results: Array<{
      item_id: string;
      payout_id: string;
      success: boolean;
      error?: string;
    }> = [];
    for (const item of validItems) {
      const payout_id = item.payout_id!;
      if (processedPayouts.has(payout_id)) {
        results.push({
          item_id: item.item_id || item.id,
          payout_id,
          success: true,
        });
      } else {
        // Check current status for error message
        try {
          const payout = await this.payoutService.findById(payout_id);
          results.push({
            item_id: item.item_id || item.id,
            payout_id,
            success: false,
            error: payout
              ? `Payout status is ${payout.status}, not scanned`
              : 'Payout not found',
          });
        } catch (error) {
          results.push({
            item_id: item.item_id || item.id,
            payout_id,
            success: false,
            error: `Internal server error: ${error}`,
          });
        }
      }
    }

    return results;
  }

  @Get('protected/orders/admin/all')
  async getAllOrdersAdmin() {
    return await this.paymentService.getAllOrdersAdmin();
  }
}

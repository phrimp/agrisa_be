/* eslint-disable @typescript-eslint/no-unsafe-call */
import { Injectable, Logger } from '@nestjs/common';
import { Cron } from '@nestjs/schedule';
import { Inject } from '@nestjs/common';
import type { PaymentService } from './payment.service';

@Injectable()
export class ExpiredCheckerService {
  private readonly logger = new Logger(ExpiredCheckerService.name);

  constructor(
    @Inject('PaymentService') private readonly paymentService: PaymentService,
  ) {}

  @Cron(process.env.PAYMENT_CRON_EXPRESSION || '0 6 * * *')
  async checkExpiredPayments() {
    try {
      const expiredPayments = await this.paymentService.findExpired();

      for (const payment of expiredPayments) {
        await this.paymentService.update(payment.id, { status: 'expired' });
      }
    } catch (error) {
      this.logger.error('Lỗi kiểm tra thanh toán hết hạn', error);
    }
  }
}

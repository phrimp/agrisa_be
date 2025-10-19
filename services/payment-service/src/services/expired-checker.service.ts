import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { SchedulerRegistry } from '@nestjs/schedule';
import { CronJob } from 'cron';
import { Inject } from '@nestjs/common';
import type { PaymentService } from './payment.service';

@Injectable()
export class ExpiredCheckerService implements OnModuleInit {
  private readonly logger = new Logger(ExpiredCheckerService.name);

  constructor(
    @Inject('PaymentService') private readonly paymentService: PaymentService,
    private readonly schedulerRegistry: SchedulerRegistry,
  ) {}

  onModuleInit() {
    const cronExpression = process.env.PAYMENT_CRON_EXPRESSION || '0 6 * * *';

    // Add new job with config expression
    const job = new CronJob(cronExpression, async () => {
      await this.checkExpiredPayments();
    });
    this.schedulerRegistry.addCronJob('checkExpiredPayments', job);
    job.start();
  }

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

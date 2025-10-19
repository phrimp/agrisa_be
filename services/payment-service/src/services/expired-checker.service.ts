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

    this.logger.log(`Initializing cron job with expression: ${cronExpression}`);

    try {
      // Add new job with config expression
      const job = new CronJob(cronExpression, async () => {
        await this.checkExpiredPayments();
      });
      this.schedulerRegistry.addCronJob('checkExpiredPayments', job);
      job.start();
      this.logger.log('Expired payment checker cron job started successfully');
    } catch (error) {
      this.logger.error(
        `Failed to start cron job with expression: ${cronExpression}`,
        error,
      );
      // Fallback to default safe cron expression
      this.logger.warn('Using fallback cron expression: 0 6 * * *');
      const fallbackJob = new CronJob('0 6 * * *', async () => {
        await this.checkExpiredPayments();
      });
      this.schedulerRegistry.addCronJob('checkExpiredPayments', fallbackJob);
      fallbackJob.start();
    }
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

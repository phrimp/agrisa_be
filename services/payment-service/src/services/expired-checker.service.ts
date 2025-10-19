/* eslint-disable @typescript-eslint/no-unsafe-call */
import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { SchedulerRegistry } from '@nestjs/schedule';
import { CronJob } from 'cron';
import { Inject } from '@nestjs/common';
import type { PaymentService } from './payment.service';
import type { ConfigurationService } from './configuration.service';

@Injectable()
export class ExpiredCheckerService implements OnModuleInit {
  private readonly logger = new Logger(ExpiredCheckerService.name);

  constructor(
    @Inject('PaymentService') private readonly paymentService: PaymentService,
    @Inject('ConfigurationService')
    private readonly configService: ConfigurationService,
    private readonly schedulerRegistry: SchedulerRegistry,
  ) {}

  async onModuleInit() {
    const config = await this.configService.getConfiguration();
    const cronExpression = config?.payment_cron_expression || '0 6 * * *';

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

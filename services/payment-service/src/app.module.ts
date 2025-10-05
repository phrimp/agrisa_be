import { Module } from '@nestjs/common';
import { ScheduleModule } from '@nestjs/schedule';
import { PingController } from './controllers/ping.controller';
import { PaymentController } from './controllers/payment.controller';
import { ImplPingService } from './services/impl.ping.service';
import { ImplPayosService } from './services/impl.payos.service';
import { ImplPaymentService } from './services/impl.payment.service';
import { ExpiredCheckerService } from './services/expired-checker.service';
import { TypeOrmModule } from '@nestjs/typeorm';
import { databaseConfig } from './libs/db.config';
import { Payment } from './entities/payment.entity';
import { PaymentRepository } from './repositories/payment.repository';

@Module({
  imports: [
    ScheduleModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Payment]),
  ],
  controllers: [PingController, PaymentController],
  providers: [
    PaymentRepository,
    ExpiredCheckerService,
    { provide: 'PingService', useClass: ImplPingService },
    { provide: 'PayosService', useClass: ImplPayosService },
    { provide: 'PaymentService', useClass: ImplPaymentService },
  ],
})
export class AppModule {}

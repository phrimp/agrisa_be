import { Module } from '@nestjs/common';
import { ScheduleModule } from '@nestjs/schedule';
import { TypeOrmModule } from '@nestjs/typeorm';
import { PaymentController } from './controllers/payment.controller';
import { PingController } from './controllers/ping.controller';
import { OrderItem } from './entities/order-item.entity';
import { Payment } from './entities/payment.entity';
import { Payout } from './entities/payout.entity';
import { databaseConfig } from './libs/db.config';
import { OrderItemRepository } from './repositories/order-item.repository';
import { PaymentRepository } from './repositories/payment.repository';
import { PayoutRepository } from './repositories/payout.repository';
import { ExpiredCheckerService } from './services/expired-checker.service';
import { ImplOrderItemService } from './services/impl.order-item.service';
import { ImplPaymentService } from './services/impl.payment.service';
import { ImplPayosService } from './services/impl.payos.service';
import { ImplPayoutService } from './services/impl.payout.service';
import { ImplPingService } from './services/impl.ping.service';

@Module({
  imports: [
    ScheduleModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Payment, OrderItem, Payout]),
  ],
  controllers: [PingController, PaymentController],
  providers: [
    PaymentRepository,
    OrderItemRepository,
    PayoutRepository,
    ExpiredCheckerService,
    { provide: 'PingService', useClass: ImplPingService },
    { provide: 'PayosService', useClass: ImplPayosService },
    { provide: 'PaymentService', useClass: ImplPaymentService },
    { provide: 'OrderItemService', useClass: ImplOrderItemService },
    { provide: 'PayoutService', useClass: ImplPayoutService },
  ],
})
export class AppModule {}

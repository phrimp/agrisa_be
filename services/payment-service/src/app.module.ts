import { Module } from '@nestjs/common';
import { ScheduleModule } from '@nestjs/schedule';
import { TypeOrmModule } from '@nestjs/typeorm';
import { PaymentController } from './controllers/payment.controller';
import { PingController } from './controllers/ping.controller';
import { Item } from './entities/item.entity';
import { Payment } from './entities/payment.entity';
import { Payout } from './entities/payout.entity';
import { databaseConfig } from './libs/db.config';
import { ItemRepository } from './repositories/item.repository';
import { PaymentRepository } from './repositories/payment.repository';
import { PayoutRepository } from './repositories/payout.repository';
import { ExpiredCheckerService } from './services/expired-checker.service';
import { ImplItemService } from './services/impl.item.service';
import { ImplPaymentService } from './services/impl.payment.service';
import { ImplPayosService } from './services/impl.payos.service';
import { ImplPayoutService } from './services/impl.payout.service';
import { ImplPingService } from './services/impl.ping.service';

@Module({
  imports: [
    ScheduleModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Payment, Item, Payout]),
  ],
  controllers: [PingController, PaymentController],
  providers: [
    PaymentRepository,
    ItemRepository,
    PayoutRepository,
    ExpiredCheckerService,
    { provide: 'PingService', useClass: ImplPingService },
    { provide: 'PayosService', useClass: ImplPayosService },
    { provide: 'PaymentService', useClass: ImplPaymentService },
    { provide: 'ItemService', useClass: ImplItemService },
    { provide: 'PayoutService', useClass: ImplPayoutService },
  ],
})
export class AppModule {}

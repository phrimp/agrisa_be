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
import { ConfigurationController } from './controllers/configuration.controller';
import { ImplConfigurationService } from './services/impl.configuration.service';
import { Configuration } from './entities/configuration.entity';

@Module({
  imports: [
    ScheduleModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Payment, Configuration]),
  ],
  controllers: [PingController, PaymentController, ConfigurationController],
  providers: [
    PaymentRepository,
    ExpiredCheckerService,
    { provide: 'PingService', useClass: ImplPingService },
    { provide: 'PayosService', useClass: ImplPayosService },
    { provide: 'PaymentService', useClass: ImplPaymentService },
    { provide: 'ConfigurationService', useClass: ImplConfigurationService },
  ],
})
export class AppModule {}

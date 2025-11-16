import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { NotiController } from './controllers/noti.controller';
import { Subscriber } from './entities/subscriber.entity';
import { ImplSubscriberService } from './services/impl.subscriber.service';
import { ImplPushNotificationService } from './services/impl.push-notification.service';
import { PushNotificationConsumer } from './events/consumers';
import { databaseConfig } from './libs/db.config';

@Module({
  imports: [
    ConfigModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Subscriber]),
  ],
  controllers: [AppController, NotiController],
  providers: [
    AppService,
    {
      provide: 'ISubscriberService',
      useClass: ImplSubscriberService,
    },
    {
      provide: 'IPushNotificationService',
      useClass: ImplPushNotificationService,
    },
    PushNotificationConsumer,
  ],
})
export class AppModule {}

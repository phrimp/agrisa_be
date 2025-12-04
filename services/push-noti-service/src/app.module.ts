import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { NotiController } from './controllers/noti.controller';
import { Notification } from './entities/notification.entity';
import { Subscriber } from './entities/subscriber.entity';
import { PushNotificationConsumer } from './events/consumers';
import { databaseConfig } from './libs/db.config';
import { ImplNotificationService } from './services/impl.notification.service';
import { ImplPushNotificationService } from './services/impl.push-notification.service';
import { ImplSubscriberService } from './services/impl.subscriber.service';

@Module({
  imports: [
    ConfigModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Subscriber, Notification]),
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
    {
      provide: 'INotificationService',
      useClass: ImplNotificationService,
    },
    PushNotificationConsumer,
  ],
})
export class AppModule {}

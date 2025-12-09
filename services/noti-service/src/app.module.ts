import { PushNotiController } from '@/controllers/push-noti.controller';
import { Notification } from '@/entities/notification.entity';
import { Receiver } from '@/entities/receiver.entity';
import { Subscriber } from '@/entities/subcriber.entity';
import { databaseConfig } from '@/libs/db.config';
import { NotificationGateway } from '@/services/notification.gateway';
import { PushNotiService } from '@/services/push-noti.service';
import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';

@Module({
  imports: [
    ConfigModule.forRoot(),
    TypeOrmModule.forRoot(databaseConfig),
    TypeOrmModule.forFeature([Notification, Receiver, Subscriber]),
  ],
  controllers: [PushNotiController],
  providers: [PushNotiService, NotificationGateway],
})
export class AppModule {}

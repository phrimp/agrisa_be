/* eslint-disable @typescript-eslint/no-unsafe-assignment */
/* eslint-disable @typescript-eslint/no-unsafe-return */
import { Controller, Post, Body, Inject } from '@nestjs/common';
import type { IPushNotificationService } from 'src/services/push-notification.service';
import type { ISubscriberService } from 'src/services/subscriber.service';

@Controller('push-noti/public')
export class NotiController {
  constructor(
    @Inject('IPushNotificationService')
    private readonly pushNotificationService: IPushNotificationService,
    @Inject('ISubscriberService')
    private readonly subscriberService: ISubscriberService,
  ) {}

  // Đăng ký Expo token
  @Post('subscribe/expo')
  async subscribeExpo(@Body() body: { expo_token: string; user_id: string }) {
    return await this.subscriberService.registerSubscriber({
      expo_token: body.expo_token,
      user_id: body.user_id,
      type: 'expo',
    });
  }

  // Đăng ký Web Push subscription
  @Post('subscribe/web')
  async subscribeWeb(
    @Body()
    body: {
      user_id: string;
      endpoint: string;
      keys: { p256dh: string; auth: string };
    },
  ) {
    return await this.subscriberService.registerSubscriber({
      user_id: body.user_id,
      type: 'web',
      endpoint: body.endpoint,
      p256dh: body.keys.p256dh,
      auth: body.keys.auth,
    });
  }

  // Gửi notification cho một user cụ thể
  @Post('send/user')
  async sendToUser(
    @Body()
    body: {
      user_id: string;
      title: string;
      body: string;
      data?: any;
    },
  ) {
    await this.pushNotificationService.sendToUser(body.user_id, {
      title: body.title,
      body: body.body,
      data: body.data,
    });
    return { success: true, message: 'Notification sent' };
  }

  // Gửi notification cho nhiều users
  @Post('send/users')
  async sendToUsers(
    @Body()
    body: {
      user_ids: string[];
      title: string;
      body: string;
      data?: any;
    },
  ) {
    await this.pushNotificationService.sendToMultipleUsers(body.user_ids, {
      title: body.title,
      body: body.body,
      data: body.data,
    });
    return { success: true, message: 'Notifications sent' };
  }

  // Gửi notification cho tất cả users
  @Post('send/all')
  async sendToAll(
    @Body()
    body: {
      title: string;
      body: string;
      data?: any;
    },
  ) {
    await this.pushNotificationService.sendToAll({
      title: body.title,
      body: body.body,
      data: body.data,
    });
    return { success: true, message: 'Notifications sent to all users' };
  }
}

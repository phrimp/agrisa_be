/* eslint-disable @typescript-eslint/no-unsafe-assignment */
/* eslint-disable @typescript-eslint/no-unsafe-return */
import {
  Body,
  Controller,
  Delete,
  Get,
  Headers,
  Inject,
  Param,
  Patch,
  Post,
  Query,
} from '@nestjs/common';
import type { INotificationService } from 'src/services/notification.service';
import type { IPushNotificationService } from 'src/services/push-notification.service';
import type { ISubscriberService } from 'src/services/subscriber.service';

@Controller('push-noti')
export class NotiController {
  constructor(
    @Inject('IPushNotificationService')
    private readonly pushNotificationService: IPushNotificationService,
    @Inject('ISubscriberService')
    private readonly subscriberService: ISubscriberService,
    @Inject('INotificationService')
    private readonly notificationService: INotificationService,
  ) {}

  @Get('/protected/permission')
  getPermission(@Headers('x-user-id') user_id: string) {
    return {
      return_url: `https://agrisa-noti.phrimp.io.vn?user_id=${user_id}`,
    };
  }

  // Đăng ký Expo token
  @Post('/public/subscribe/expo')
  async subscribeExpo(@Body() body: { expo_token: string; user_id: string }) {
    return await this.subscriberService.registerSubscriber({
      expo_token: body.expo_token,
      user_id: body.user_id,
      type: 'expo',
    });
  }

  // Đăng ký Web Push subscription
  @Post('/public/subscribe/web')
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
  @Post('/public/send/user')
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
  @Post('/public/send/users')
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
  @Post('/public/send/all')
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

  // Unsubscribe - Hủy đăng ký notification
  @Delete('/public/unsubscribe')
  async unsubscribe(@Body() body: { user_id: string; type: 'expo' | 'web' }) {
    await this.subscriberService.unsubscribe(body.user_id, body.type);
    return { success: true, message: 'Unsubscribed successfully' };
  }

  // Lấy lịch sử notification của user
  @Get('/protected/history')
  async getHistory(
    @Headers('x-user-id') user_id: string,
    @Query('page') page?: string,
    @Query('limit') limit?: string,
  ) {
    const result = await this.notificationService.getNotificationsByUserId(
      user_id,
      page ? parseInt(page) : 1,
      limit ? parseInt(limit) : 20,
    );
    return { success: true, ...result };
  }

  // Đánh dấu notification đã đọc
  @Patch('/protected/read/:id')
  async markAsRead(@Param('id') id: string) {
    await this.notificationService.markAsRead(id);
    return { success: true, message: 'Marked as read' };
  }

  // Đếm số notification chưa đọc
  @Get('/protected/unread-count')
  async getUnreadCount(@Headers('x-user-id') user_id: string) {
    const count = await this.notificationService.getUnreadCount(user_id);
    return { success: true, count };
  }
}

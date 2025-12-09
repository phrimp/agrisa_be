import { Notification } from '@/entities/notification.entity';
import { Receiver } from '@/entities/receiver.entity';
import { Subscriber } from '@/entities/subcriber.entity';
import { ePlatform } from '@/libs/enum';
import { SendPayloadDto } from '@/libs/types/send-payload.dto';
import { SubscribeDto } from '@/libs/types/subscribe.dto';
import { setupWebPush } from '@/libs/web-push.config';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Expo, ExpoPushMessage } from 'expo-server-sdk';
import { In, Repository } from 'typeorm';
import * as webpush from 'web-push';
import { NotificationGateway } from './notification.gateway';

@Injectable()
export class PushNotiService {
  private webpush: typeof webpush;
  private expo: Expo;

  constructor(
    @InjectRepository(Notification)
    private readonly notificationRepository: Repository<Notification>,
    @InjectRepository(Receiver)
    private readonly receiverRepository: Repository<Receiver>,
    @InjectRepository(Subscriber)
    private readonly subscriberRepository: Repository<Subscriber>,
    private readonly notificationGateway: NotificationGateway,
  ) {
    this.webpush = setupWebPush();
    this.expo = new Expo();
  }

  async send(data: SendPayloadDto) {
    const notification = await this.notificationRepository.save({
      title: data.title,
      body: data.body,
      data: data.data ?? null,
    });

    await this.createReceivers(notification.id, data.lstUserIds || []);

    await Promise.all([
      this.sendWeb(data),
      this.sendAndroid(data),
      this.sendIOS(data, notification.id),
    ]);

    return {
      message: 'Notification đã được gửi',
      notificationId: notification.id,
    };
  }

  private async createReceivers(notificationId: string, userIds: string[]) {
    const where: any = {};
    if (userIds && userIds.length > 0) {
      where.user_id = In(userIds);
    }

    const subscribers = await this.subscriberRepository.find({ where });

    const receivers = subscribers.map(sub => ({
      notification_id: notificationId,
      user_id: sub.user_id,
      platform: sub.platform,
      status: 'sent',
    }));

    if (receivers.length > 0) {
      await this.receiverRepository.save(receivers);
    }
  }

  async sendWeb(data: SendPayloadDto) {
    const notification = await this.notificationRepository.save({
      title: data.title,
      body: data.body,
      data: data.data ?? null,
    });

    const where: any = { platform: ePlatform.web };
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      where.user_id = In(data.lstUserIds);
    }

    const subscribers = await this.subscriberRepository.find({
      where,
    });

    const receivers = subscribers.map(sub => ({
      notification_id: notification.id,
      user_id: sub.user_id,
      platform: ePlatform.web,
      status: 'sent',
    }));

    if (receivers.length > 0) {
      await this.receiverRepository.save(receivers);
    }

    const payload = JSON.stringify({
      title: data.title,
      body: data.body,
    });

    for (const sub of subscribers) {
      if (sub.endpoint && sub.p256dh && sub.auth) {
        try {
          await this.webpush.sendNotification(
            {
              endpoint: sub.endpoint,
              keys: {
                p256dh: sub.p256dh,
                auth: sub.auth,
              },
            },
            payload,
          );
        } catch (error) {
          console.error(`Lỗi: `, error);
        }
      }
    }

    return {
      message: 'Notification đã được gửi cho Web',
      notificationId: notification.id,
    };
  }

  async sendAndroid(data: SendPayloadDto) {
    const notification = await this.notificationRepository.save({
      title: data.title,
      body: data.body,
      data: data.data ?? null,
    });

    const where: any = { platform: ePlatform.android };
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      where.user_id = In(data.lstUserIds);
    }

    const subscribers = await this.subscriberRepository.find({
      where,
    });

    const receivers = subscribers.map(sub => ({
      notification_id: notification.id,
      user_id: sub.user_id,
      platform: ePlatform.android,
      status: 'sent',
    }));

    if (receivers.length > 0) {
      await this.receiverRepository.save(receivers);
    }

    const messages: ExpoPushMessage[] = [];

    for (const sub of subscribers) {
      if (sub.expo_token && Expo.isExpoPushToken(sub.expo_token)) {
        messages.push({
          to: sub.expo_token,
          title: data.title,
          body: data.body,
        });
      }
    }

    const chunks = this.expo.chunkPushNotifications(messages);

    for (const chunk of chunks) {
      try {
        await this.expo.sendPushNotificationsAsync(chunk);
      } catch (error) {
        console.error('Lỗi: ', error);
      }
    }

    return {
      message: 'Notification đã được gửi cho Android',
      notificationId: notification.id,
    };
  }

  async sendIOS(data: SendPayloadDto, notificationId?: string) {
    const notification = notificationId
      ? { id: notificationId, created_at: new Date() }
      : await this.notificationRepository.save({
          title: data.title,
          body: data.body,
          data: data.data ?? null,
        });

    const where: any = { platform: ePlatform.ios };
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      where.user_id = In(data.lstUserIds);
    }

    const subscribers = await this.subscriberRepository.find({
      where,
    });

    if (!notificationId) {
      const receivers = subscribers.map(sub => ({
        notification_id: notification.id,
        user_id: sub.user_id,
        platform: ePlatform.ios,
        status: 'sent',
      }));

      if (receivers.length > 0) {
        await this.receiverRepository.save(receivers);
      }
    }

    const payload = {
      type: 'notification',
      id: notification.id,
      title: data.title,
      body: data.body,
      data: data.data,
      createdAt: notification.created_at,
    };

    let onlineCount = 0;
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      data.lstUserIds.forEach(userId => {
        if (this.notificationGateway.sendToUser(userId, payload)) {
          onlineCount++;
        }
      });
    } else {
      subscribers.forEach(sub => {
        if (this.notificationGateway.sendToUser(sub.user_id, payload)) {
          onlineCount++;
        }
      });
    }

    return {
      message: 'Notification đã được tạo cho iOS (WebSocket + Pull)',
      notificationId: notification.id,
      stats: {
        totalSubscribers: subscribers.length,
        sentViaWebSocket: onlineCount,
        availableForPull: subscribers.length,
      },
    };
  }

  async subscribeWeb(userId: string, data: SubscribeDto) {
    const existing = await this.subscriberRepository.findOne({
      where: { user_id: userId, platform: ePlatform.web },
    });

    if (!existing) {
      const newSub = this.subscriberRepository.create({
        user_id: userId,
        platform: ePlatform.web,
        endpoint: data.endpoint,
        p256dh: data.p256dh,
        auth: data.auth,
      });
      await this.subscriberRepository.save(newSub);
      return {
        message: 'Đăng ký nhận thông báo thành công',
        data: newSub,
      };
    }
  }

  async subscribeAndroid(userId: string, data: SubscribeDto) {
    const existing = await this.subscriberRepository.findOne({
      where: { user_id: userId, platform: ePlatform.android },
    });

    if (!existing) {
      const newSub = this.subscriberRepository.create({
        user_id: userId,
        platform: ePlatform.android,
        expo_token: data.expoToken,
      });
      await this.subscriberRepository.save(newSub);
      return {
        message: 'Đăng ký nhận thông báo thành công',
        data: newSub,
      };
    }
  }

  async subscribeIOS(userId: string) {
    const existing = await this.subscriberRepository.findOne({
      where: { user_id: userId, platform: ePlatform.ios },
    });

    if (!existing) {
      const newSub = this.subscriberRepository.create({
        user_id: userId,
        platform: ePlatform.ios,
      });
      await this.subscriberRepository.save(newSub);
      return {
        message: 'Đăng ký nhận thông báo thành công',
        data: newSub,
      };
    }
  }

  async unsubscribeWeb(userId: string) {
    await this.subscriberRepository.delete({
      user_id: userId,
      platform: ePlatform.web,
    });
    return { message: 'Hủy đăng ký nhận thông báo thành công' };
  }

  async unsubscribeAndroid(userId: string) {
    await this.subscriberRepository.delete({
      user_id: userId,
      platform: ePlatform.android,
    });
    return { message: 'Hủy đăng ký nhận thông báo thành công' };
  }

  async unsubscribeIOS(userId: string) {
    await this.subscriberRepository.delete({
      user_id: userId,
      platform: ePlatform.ios,
    });
    return { message: 'Hủy đăng ký nhận thông báo thành công' };
  }

  async isSubcribed(userId: string, platform: string) {
    const existing = await this.subscriberRepository.findOne({
      where: { user_id: userId, platform },
    });
    return {
      value: !!existing,
    };
  }

  async pullNotifications(userId: string, limit: number = 10) {
    const receivers = await this.receiverRepository.find({
      where: {
        user_id: userId,
        platform: ePlatform.ios,
        status: 'sent',
      },
      relations: ['notification'],
      order: {
        created_at: 'DESC',
      },
      take: limit,
    });

    const hasNew = receivers.length > 0;
    const notifications = receivers.map(r => ({
      id: r.id,
      notificationId: r.notification_id,
      title: r.notification.title,
      body: r.notification.body,
      data: r.notification.data,
      createdAt: r.notification.created_at,
    }));

    if (hasNew) {
      const receiverIds = receivers.map(r => r.id);
      await this.markAsRead(receiverIds);
    }

    return {
      hasNew,
      count: receivers.length,
      notifications,
    };
  }

  async markAsRead(receiverIds: string[]) {
    await this.receiverRepository.update(
      {
        id: In(receiverIds),
      },
      {
        status: 'read',
      },
    );

    return {
      message: 'Đã đánh dấu thông báo đã đọc',
      count: receiverIds.length,
    };
  }

  async getAllNotifications(userId: string, page: number = 1, limit: number = 20, status?: string) {
    const skip = (page - 1) * limit;

    const where: any = {
      user_id: userId,
    };

    if (status) {
      where.status = status;
    }

    const [receivers, total] = await this.receiverRepository.findAndCount({
      where,
      relations: ['notification'],
      order: {
        created_at: 'DESC',
      },
      skip,
      take: limit,
    });

    const unread = await this.receiverRepository.count({
      where: {
        user_id: userId,
        status: 'sent',
      },
    });

    const notifications = receivers.map(r => ({
      id: r.id,
      notificationId: r.notification_id,
      title: r.notification.title,
      body: r.notification.body,
      data: r.notification.data,
      platform: r.platform,
      status: r.status,
      createdAt: r.notification.created_at,
    }));

    return {
      data: notifications,
      unread,
      pagination: {
        page,
        limit,
        total,
        totalPages: Math.ceil(total / limit),
      },
    };
  }
}

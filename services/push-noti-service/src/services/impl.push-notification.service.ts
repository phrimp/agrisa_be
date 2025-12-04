/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/no-unnecessary-type-assertion */
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
/* eslint-disable @typescript-eslint/restrict-template-expressions */
import { Inject, Injectable, Logger } from '@nestjs/common';
import { Expo, ExpoPushMessage } from 'expo-server-sdk';
import * as webpush from 'web-push';
import type { Subscriber } from '../entities/subscriber.entity';
import type { INotificationService } from './notification.service';
import type {
  IPushNotificationService,
  NotificationPayload,
} from './push-notification.service';
import type { ISubscriberService } from './subscriber.service';

@Injectable()
export class ImplPushNotificationService implements IPushNotificationService {
  private readonly logger = new Logger(ImplPushNotificationService.name);
  private expo: Expo;

  constructor(
    @Inject('ISubscriberService')
    private readonly subscriberService: ISubscriberService,
    @Inject('INotificationService')
    private readonly notificationService: INotificationService,
  ) {
    this.expo = new Expo();
    this.initializeWebPush();
  }

  private initializeWebPush() {
    // Tạo VAPID keys nếu chưa có
    // Chạy: npx web-push generate-vapid-keys
    const vapidPublicKey =
      process.env.VAPID_PUBLIC_KEY || 'YOUR_VAPID_PUBLIC_KEY';
    const vapidPrivateKey =
      process.env.VAPID_PRIVATE_KEY || 'YOUR_VAPID_PRIVATE_KEY';
    const vapidSubject = process.env.VAPID_SUBJECT || 'mailto:your@email.com';

    webpush.setVapidDetails(vapidSubject, vapidPublicKey, vapidPrivateKey);
  }

  async send(
    notification: NotificationPayload,
    user_id?: string[],
  ): Promise<void> {
    if (user_id && user_id.length > 0) {
      await this.sendToMultipleUsers(user_id, notification);
    } else {
      await this.sendToAll(notification);
    }
  }

  async sendToUser(
    userId: string,
    notification: NotificationPayload,
  ): Promise<void> {
    const subscribers =
      await this.subscriberService.getSubscribersByUserId(userId);

    if (!subscribers || subscribers.length === 0) {
      this.logger.warn(`No subscribers found for user ${userId}`);
      return;
    }

    const promises = subscribers.map(async (subscriber: Subscriber) => {
      try {
        if (subscriber.type === 'expo') {
          await this.sendExpoNotification(subscriber.expo_token, notification);
        } else if (subscriber.type === 'web') {
          await this.sendWebPushNotification(subscriber, notification);
        }

        // Lưu thành công vào DB
        await this.notificationService.createNotification({
          user_id: userId,
          title: notification.title,
          body: notification.body,
          data: notification.data,
          type: subscriber.type,
          status: 'sent',
        });
      } catch (error) {
        // Lưu lỗi vào DB
        await this.notificationService.createNotification({
          user_id: userId,
          title: notification.title,
          body: notification.body,
          data: notification.data,
          type: subscriber.type,
          status: 'failed',
          error_message: (error as Error).message,
        });
      }
    });

    await Promise.allSettled(promises);
  }

  async sendToMultipleUsers(
    userIds: string[],
    notification: NotificationPayload,
  ): Promise<void> {
    const promises = userIds.map((userId) =>
      this.sendToUser(userId, notification),
    );
    await Promise.allSettled(promises);
  }

  async sendToAll(notification: NotificationPayload): Promise<void> {
    const subscribers = await this.subscriberService.getAllSubscribers();

    const expoTokens = subscribers
      .filter(
        (s: Subscriber) =>
          s.type === 'expo' && Expo.isExpoPushToken(s.expo_token),
      )
      .map((s: Subscriber) => s.expo_token);

    const webSubscribers = subscribers.filter(
      (s: Subscriber) => s.type === 'web',
    );

    // Gửi Expo notifications
    if (expoTokens.length > 0) {
      await this.sendBulkExpoNotifications(expoTokens, notification);
    }

    // Gửi Web Push notifications
    const webPromises = webSubscribers.map((subscriber) =>
      this.sendWebPushNotification(subscriber, notification),
    );
    await Promise.allSettled(webPromises);
  }

  private async sendExpoNotification(
    token: string,
    notification: NotificationPayload,
  ): Promise<void> {
    if (!Expo.isExpoPushToken(token)) {
      this.logger.error(`Invalid Expo push token: ${token}`);
      return;
    }

    const message: ExpoPushMessage = {
      to: token,
      sound: 'default',
      title: notification.title,
      body: notification.body,
      data: (notification.data as Record<string, any>) || {},
    };

    try {
      const chunks = this.expo.chunkPushNotifications([message]);
      const tickets = await this.expo.sendPushNotificationsAsync(chunks[0]);

      for (const ticket of tickets) {
        if (ticket.status === 'error') {
          this.logger.error(
            `Error sending to ${token}: ${ticket.message}`,
            ticket.details,
          );
        }
      }
    } catch (error) {
      this.logger.error(
        `Error sending Expo notification: ${(error as Error).message}`,
      );
    }
  }

  private async sendBulkExpoNotifications(
    tokens: string[],
    notification: NotificationPayload,
  ): Promise<void> {
    const messages: ExpoPushMessage[] = tokens.map((token) => ({
      to: token,
      sound: 'default',
      title: notification.title,
      body: notification.body,
      data: (notification.data as Record<string, any>) || {},
    }));

    const chunks = this.expo.chunkPushNotifications(messages);

    for (const chunk of chunks) {
      try {
        const tickets = await this.expo.sendPushNotificationsAsync(chunk);

        for (let i = 0; i < tickets.length; i++) {
          const ticket = tickets[i];
          if (ticket.status === 'error') {
            this.logger.error(
              `Error sending to ${String(chunk[i].to)}: ${ticket.message}`,
              ticket.details,
            );
          }
        }
      } catch (error) {
        this.logger.error(
          `Error sending bulk Expo notifications: ${(error as Error).message}`,
        );
      }
    }
  }

  private async sendWebPushNotification(
    subscriber: Subscriber,
    notification: NotificationPayload,
  ): Promise<void> {
    if (!subscriber.endpoint || !subscriber.p256dh || !subscriber.auth) {
      this.logger.warn(
        `Invalid web push subscription for user ${subscriber.user_id}`,
      );
      return;
    }

    const pushSubscription = {
      endpoint: subscriber.endpoint,
      keys: {
        p256dh: subscriber.p256dh,
        auth: subscriber.auth,
      },
    };

    // Format đúng cho Service Worker
    const payload = {
      title: notification.title,
      body: notification.body,
      data: notification.data || {},
    };

    try {
      await webpush.sendNotification(
        pushSubscription,
        JSON.stringify(payload),
        {
          headers: {
            Topic: 'agrisa-web', // BẮT BUỘC CHO iOS
            Urgency: 'normal',
          },
          contentEncoding: 'aes128gcm', // BẮT BUỘC CHO iOS
          TTL: 2419200,
        },
      );
      this.logger.log(
        `Web push notification sent to user ${subscriber.user_id}`,
      );
    } catch (error) {
      this.logger.error(
        `Error sending web push notification: ${(error as Error).message}`,
      );

      // Nếu subscription không còn hợp lệ (410 Gone), có thể xóa khỏi DB
      if ((error as any).statusCode === 410) {
        this.logger.warn(`Subscription expired for user ${subscriber.user_id}`);
      }
    }
  }
}

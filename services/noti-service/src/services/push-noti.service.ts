import { Notification } from '@/entities/notification.entity';
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
import WebSocket from 'ws';

@Injectable()
export class PushNotiService {
  private webpush: typeof webpush;
  private expo: Expo;

  constructor(
    @InjectRepository(Notification)
    private readonly notificationRepository: Repository<Notification>,
    @InjectRepository(Subscriber)
    private readonly subscriberRepository: Repository<Subscriber>,
  ) {
    this.webpush = setupWebPush();
    this.expo = new Expo();
  }

  async send(data: SendPayloadDto) {
    await Promise.all([this.sendWeb(data), this.sendAndroid(data), this.sendIOS(data)]);
  }

  async sendWeb(data: SendPayloadDto) {
    const where: any = { platform: ePlatform.web };
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      where.user_id = In(data.lstUserIds);
    }

    const subscribers = await this.subscriberRepository.find({
      where,
    });

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
  }

  async sendAndroid(data: SendPayloadDto) {
    const where: any = { platform: ePlatform.android };
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      where.user_id = In(data.lstUserIds);
    }

    const subscribers = await this.subscriberRepository.find({
      where,
    });

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
  }

  async sendIOS(data: SendPayloadDto) {
    const where: any = { platform: ePlatform.ios };
    if (data.lstUserIds && data.lstUserIds.length > 0) {
      where.user_id = In(data.lstUserIds);
    }

    const subscribers = await this.subscriberRepository.find({
      where,
    });

    const payload = JSON.stringify({
      title: data.title,
      body: data.body,
    });

    for (const sub of subscribers) {
      if (sub.endpoint) {
        try {
          const ws = new WebSocket(sub.endpoint);
          ws.on('open', () => {
            ws.send(payload);
            ws.close();
          });
          ws.on('error', error => {
            console.error('Lỗi: ', sub.id, error);
          });
        } catch (error) {
          console.error('Lỗi: ', sub.id, error);
        }
      }
    }
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
    }
  }

  async subscribeIOS(userId: string, data: SubscribeDto) {
    const existing = await this.subscriberRepository.findOne({
      where: { user_id: userId, platform: ePlatform.ios },
    });

    if (!existing) {
      const newSub = this.subscriberRepository.create({
        user_id: userId,
        platform: ePlatform.ios,
        endpoint: data.endpoint,
      });
      await this.subscriberRepository.save(newSub);
    }
  }

  async unsubscribeWeb(userId: string) {
    await this.subscriberRepository.delete({
      user_id: userId,
      platform: ePlatform.web,
    });
  }

  async unsubscribeAndroid(userId: string) {
    await this.subscriberRepository.delete({
      user_id: userId,
      platform: ePlatform.android,
    });
  }

  async unsubscribeIOS(userId: string) {
    await this.subscriberRepository.delete({
      user_id: userId,
      platform: ePlatform.ios,
    });
  }
}

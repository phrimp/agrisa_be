import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Subscriber } from 'src/entities/subscriber.entity';
import { Repository } from 'typeorm';
import { ISubscriberService } from './subscriber.service';

@Injectable()
export class ImplSubscriberService implements ISubscriberService {
  constructor(
    @InjectRepository(Subscriber)
    private readonly subscriberRepo: Repository<Subscriber>,
  ) {}

  async registerSubscriber(data: {
    expo_token?: string;
    user_id: string;
    type: string;
    p256dh?: string;
    auth?: string;
    endpoint?: string;
  }) {
    const { expo_token, user_id, type, p256dh, auth, endpoint } = data;

    // Tìm subscriber theo user_id và type để tránh trùng lặp
    const existing = await this.subscriberRepo.findOne({
      where: { user_id, type },
    });

    if (existing) {
      if (expo_token) existing.expo_token = expo_token;
      if (p256dh) existing.p256dh = p256dh;
      if (auth) existing.auth = auth;
      if (endpoint) existing.endpoint = endpoint;
      return await this.subscriberRepo.save(existing);
    } else {
      const newSubscriber = this.subscriberRepo.create({
        expo_token: expo_token || '',
        user_id,
        type,
        p256dh: p256dh || '',
        auth: auth || '',
        endpoint: endpoint || '',
      });
      return await this.subscriberRepo.save(newSubscriber);
    }
  }

  async updateSubscriber(data: {
    expo_token?: string;
    user_id: string;
    type: string;
    p256dh?: string;
    auth?: string;
    endpoint?: string;
  }) {
    const { expo_token, user_id, type, p256dh, auth, endpoint } = data;

    const existing = await this.subscriberRepo.findOne({
      where: { user_id, type },
    });

    if (!existing) {
      throw new Error('Chưa đăng ký subscription cho thiết bị này');
    }

    if (expo_token) existing.expo_token = expo_token;
    if (p256dh) existing.p256dh = p256dh;
    if (auth) existing.auth = auth;
    if (endpoint) existing.endpoint = endpoint;

    return await this.subscriberRepo.save(existing);
  }

  async getSubscribersByUserId(userId: string): Promise<Subscriber[]> {
    return await this.subscriberRepo.find({ where: { user_id: userId } });
  }

  async getAllSubscribers(): Promise<Subscriber[]> {
    return await this.subscriberRepo.find();
  }

  async unsubscribe(userId: string, type: string): Promise<void> {
    await this.subscriberRepo.delete({ user_id: userId, type });
  }

  async deleteSubscriber(id: string): Promise<void> {
    await this.subscriberRepo.delete(id);
  }
}

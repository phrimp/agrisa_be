import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Notification } from 'src/entities/notification.entity';
import { Repository } from 'typeorm';
import type {
  INotificationService,
  NotificationListResponse,
} from './notification.service';

@Injectable()
export class ImplNotificationService implements INotificationService {
  constructor(
    @InjectRepository(Notification)
    private readonly notificationRepo: Repository<Notification>,
  ) {}

  async createNotification(data: {
    user_id: string;
    title: string;
    body: string;
    data?: any;
    type: string;
    status?: string;
    error_message?: string;
  }): Promise<Notification> {
    const notification = this.notificationRepo.create({
      user_id: data.user_id,
      title: data.title,
      body: data.body,
      data: (data.data as Record<string, any>) || {},
      type: data.type,
      status: data.status || 'sent',
      error_message: data.error_message,
    });

    return await this.notificationRepo.save(notification);
  }

  async getNotificationsByUserId(
    userId: string,
    page: number = 1,
    limit: number = 20,
  ): Promise<NotificationListResponse> {
    const skip = (page - 1) * limit;

    const [data, total] = await this.notificationRepo.findAndCount({
      where: { user_id: userId },
      order: { created_at: 'DESC' },
      take: limit,
      skip: skip,
    });

    const totalPages = Math.ceil(total / limit);

    return {
      data,
      meta: {
        total,
        currentPage: page,
        limit,
        totalPages,
        isCanNext: page < totalPages,
        isCanBack: page > 1,
      },
    };
  }

  async markAsRead(notificationId: string): Promise<void> {
    await this.notificationRepo.update(notificationId, { status: 'read' });
  }

  async getUnreadCount(userId: string): Promise<number> {
    return await this.notificationRepo.count({
      where: { user_id: userId, status: 'sent' },
    });
  }
}

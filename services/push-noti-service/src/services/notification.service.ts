import type { Notification } from 'src/entities/notification.entity';

export interface NotificationListResponse {
  data: Notification[];
  meta: {
    total: number;
    currentPage: number;
    limit: number;
    totalPages: number;
    isCanNext: boolean;
    isCanBack: boolean;
  };
}

export interface INotificationService {
  createNotification(data: {
    user_id: string;
    title: string;
    body: string;
    data?: any;
    type: string;
    status?: string;
    error_message?: string;
  }): Promise<Notification>;

  getNotificationsByUserId(
    userId: string,
    page?: number,
    limit?: number,
  ): Promise<NotificationListResponse>;

  markAsRead(notificationId: string): Promise<void>;

  getUnreadCount(userId: string): Promise<number>;
}

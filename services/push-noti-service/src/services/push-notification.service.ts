export interface IPushNotificationService {
  sendToUser(userId: string, notification: NotificationPayload): Promise<void>;
  sendToMultipleUsers(
    userIds: string[],
    notification: NotificationPayload,
  ): Promise<void>;
  sendToAll(notification: NotificationPayload): Promise<void>;
  send(notification: NotificationPayload, user_id?: string[]): Promise<void>;
}

export interface NotificationPayload {
  title: string;
  body: string;
  data?: any;
}

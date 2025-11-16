export class SubscribeExpoDto {
  expo_token: string;
  user_id: string;
}

export class SubscribeWebDto {
  user_id: string;
  endpoint: string;
  keys: {
    p256dh: string;
    auth: string;
  };
}

export class SendNotificationDto {
  user_id: string;
  title: string;
  body: string;
  data?: Record<string, any>;
}

export class SendBulkNotificationDto {
  user_ids: string[];
  title: string;
  body: string;
  data?: Record<string, any>;
}

export class BroadcastNotificationDto {
  title: string;
  body: string;
  data?: Record<string, any>;
}

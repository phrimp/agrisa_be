# Push Notification Service

Service hỗ trợ gửi push notifications cho cả **Web Push** và **Expo App**.

## Setup

### 1. Cài đặt dependencies

```bash
npm install expo-server-sdk
```

### 2. Tạo VAPID keys cho Web Push

```bash
npx web-push generate-vapid-keys
```

Copy kết quả vào file `.env`:

```env
VAPID_PUBLIC_KEY=your_public_key
VAPID_PRIVATE_KEY=your_private_key
VAPID_SUBJECT=mailto:your@email.com
```

### 3. Cấu hình Database

File `.env`:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_NAME=push_noti
```

### 4. Chạy migration để tạo bảng

```sql
-- File db.sql đã có sẵn schema cho table subscribers
```

## API Endpoints

### 1. Đăng ký Expo Token (Mobile App)

```http
POST /pushed-noti/subscribe/expo
Content-Type: application/json

{
  "expo_token": "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
  "user_id": "user123"
}
```

### 2. Đăng ký Web Push (Web Browser)

```http
POST /pushed-noti/subscribe/web
Content-Type: application/json

{
  "user_id": "user123",
  "endpoint": "https://fcm.googleapis.com/fcm/send/...",
  "keys": {
    "p256dh": "BNc...",
    "auth": "tBH..."
  }
}
```

### 3. Gửi notification cho 1 user

```http
POST /pushed-noti/send/user
Content-Type: application/json

{
  "user_id": "user123",
  "title": "Thông báo mới",
  "body": "Bạn có tin nhắn mới",
  "data": {
    "type": "message",
    "id": "msg123"
  }
}
```

### 4. Gửi notification cho nhiều users

```http
POST /pushed-noti/send/users
Content-Type: application/json

{
  "user_ids": ["user123", "user456", "user789"],
  "title": "Thông báo hệ thống",
  "body": "Hệ thống sẽ bảo trì vào 2h sáng",
  "data": {
    "type": "maintenance"
  }
}
```

### 5. Gửi broadcast notification (tất cả users)

```http
POST /pushed-noti/send/all
Content-Type: application/json

{
  "title": "Cập nhật mới",
  "body": "Phiên bản 2.0 đã có sẵn",
  "data": {
    "version": "2.0"
  }
}
```

## Client Implementation

### Web Push (Browser)

```javascript
// Request permission và đăng ký
async function registerWebPush() {
  // Request permission
  const permission = await Notification.requestPermission();
  if (permission !== 'granted') {
    console.log('Permission denied');
    return;
  }

  // Register service worker
  const registration = await navigator.serviceWorker.register('/sw.js');

  // Subscribe
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: 'YOUR_VAPID_PUBLIC_KEY', // Lấy từ server
  });

  // Gửi subscription lên server
  await fetch('/pushed-noti/subscribe/web', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      user_id: 'user123',
      endpoint: subscription.endpoint,
      keys: {
        p256dh: btoa(
          String.fromCharCode(...new Uint8Array(subscription.getKey('p256dh'))),
        ),
        auth: btoa(
          String.fromCharCode(...new Uint8Array(subscription.getKey('auth'))),
        ),
      },
    }),
  });
}
```

Service Worker (`sw.js`):

```javascript
self.addEventListener('push', function (event) {
  const data = event.data.json();

  const options = {
    body: data.body,
    icon: '/icon.png',
    badge: '/badge.png',
    data: data.data,
  };

  event.waitUntil(self.registration.showNotification(data.title, options));
});

self.addEventListener('notificationclick', function (event) {
  event.notification.close();
  event.waitUntil(clients.openWindow('/'));
});
```

### Expo App (React Native)

```javascript
import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';

// Configure notifications
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

// Register for push notifications
async function registerForPushNotifications() {
  let token;

  if (Platform.OS === 'android') {
    await Notifications.setNotificationChannelAsync('default', {
      name: 'default',
      importance: Notifications.AndroidImportance.MAX,
    });
  }

  const { status: existingStatus } = await Notifications.getPermissionsAsync();
  let finalStatus = existingStatus;

  if (existingStatus !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }

  if (finalStatus !== 'granted') {
    console.log('Permission denied');
    return;
  }

  token = (await Notifications.getExpoPushTokenAsync()).data;

  // Gửi token lên server
  await fetch('http://your-server.com/pushed-noti/subscribe/expo', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      expo_token: token,
      user_id: 'user123',
    }),
  });

  return token;
}

// Listen for notifications
useEffect(() => {
  const subscription = Notifications.addNotificationReceivedListener(
    (notification) => {
      console.log('Notification received:', notification);
    },
  );

  const responseSubscription =
    Notifications.addNotificationResponseReceivedListener((response) => {
      console.log('Notification clicked:', response);
    });

  return () => {
    subscription.remove();
    responseSubscription.remove();
  };
}, []);
```

## Database Schema

Table `subscribers`:

```sql
CREATE TABLE subscribers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  expo_token VARCHAR(255) NULL,
  type VARCHAR(255) NOT NULL, -- 'expo' hoặc 'web'
  p256dh VARCHAR(255) NULL, -- Web push key
  auth VARCHAR(500) NULL, -- Web push auth
  endpoint VARCHAR(500) NULL, -- Web push endpoint
  user_id VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_subscribers_user_id ON subscribers(user_id);
CREATE INDEX idx_subscribers_type ON subscribers(type);
```

## Features

- ✅ Gửi push notification cho Expo mobile app
- ✅ Gửi web push notification cho browser
- ✅ Hỗ trợ gửi cho 1 user, nhiều users, hoặc broadcast
- ✅ Tự động detect loại thiết bị (web/expo)
- ✅ Custom data payload
- ✅ Error handling và logging
- ✅ Batch processing cho performance

## Notes

- Mỗi user có thể có nhiều subscriptions (nhiều thiết bị)
- Type `expo` cho mobile app, `web` cho browser
- VAPID keys cần được generate và lưu an toàn
- Service tự động xử lý expired subscriptions

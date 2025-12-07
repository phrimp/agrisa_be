import * as webpush from 'web-push';

export const setupWebPush = () => {
  const vapidKeys = {
    publicKey: process.env.VAPID_PUBLIC_KEY,
    privateKey: process.env.VAPID_PRIVATE_KEY,
    vapidSubject: process.env.VAPID_SUBJECT,
  };

  if (!vapidKeys.publicKey || !vapidKeys.privateKey) {
    throw new Error('Thiếu biến môi trường VAPID_PUBLIC_KEY hoặc VAPID_PRIVATE_KEY');
  }

  webpush.setVapidDetails(vapidKeys.vapidSubject, vapidKeys.publicKey, vapidKeys.privateKey);

  return webpush;
};

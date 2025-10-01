import { PayOS } from '@payos/node';

export const payOS = new PayOS({
  clientId: process.env.PAYOS_CLIENT_ID,
  apiKey: process.env.PAYOS_API_KEY,
  checksumKey: process.env.PAYOS_CHECKSUM_KEY,
});

export const PAYOS_EXPIRED_DURATION = process.env.PAYOS_EXPIRED_DURATION;

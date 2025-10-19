export const payosConfig = {
  clientId: process.env.PAYOS_CLIENT_ID || '',
  apiKey: process.env.PAYOS_API_KEY || '',
  checksumKey: process.env.PAYOS_CHECKSUM_KEY || '',
  expiredDuration: process.env.PAYOS_EXPIRED_DURATION || '3600',
  orderCodeLength: parseInt(process.env.PAYOS_ORDER_CODE_LENGTH || '8', 10),
};

export const validatePayosConfig = () => {
  const { clientId, apiKey, checksumKey } = payosConfig;

  if (!clientId || !apiKey || !checksumKey) {
    throw new Error(
      'Thiếu biến môi trường PAYOS_CLIENT_ID, PAYOS_API_KEY hoặc PAYOS_CHECKSUM_KEY',
    );
  }

  return true;
};

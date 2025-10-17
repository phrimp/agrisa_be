/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/no-unsafe-call */
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
import amqp from 'amqplib';

export const connectRabbitMQ = async (): Promise<{
  connection: amqp.Connection;
  channel: amqp.Channel;
}> => {
  const url = process.env.RABBITMQ_URL;
  if (!url) {
    throw new Error('Chưa có biến môi trường RABBITMQ_URL');
  }

  try {
    const connection = await amqp.connect(url);
    const channel = await connection.createChannel();
    return { connection, channel };
  } catch (error) {
    console.error('Không thể kết nối tới RabbitMQ:', error);
    throw new Error('Không thể kết nối tới RabbitMQ');
  }
};

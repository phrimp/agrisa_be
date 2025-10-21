import { connectRabbitMQ } from 'src/libs/rabbitmq.config';

export const publisher = async (data) => {
  const queue = 'payment_events';
  const { connection, channel } = await connectRabbitMQ();
  await channel.assertQueue(queue, {
    durable: true,
    autoDelete: false,
    exclusive: false,
    arguments: null,
  });
  channel.sendToQueue(queue, Buffer.from(JSON.stringify(data)), {
    persistent: true,
  });
  await channel.close();
  await connection.close();
};

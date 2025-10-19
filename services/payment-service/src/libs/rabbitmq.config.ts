import amqp from 'amqplib';

export const connectRabbitMQ = async (): Promise<{
  connection: amqp.Connection;
  channel: amqp.Channel;
}> => {
  const host = process.env.RABBITMQ_HOST || 'localhost';
  const port = process.env.RABBITMQ_PORT;
  const user = process.env.RABBITMQ_USER;
  const password = process.env.RABBITMQ_PASSWORD;

  if (!port || !user || !password) {
    throw new Error(
      'Thiếu biến môi trường RABBITMQ_PORT, RABBITMQ_USER hoặc RABBITMQ_PASSWORD',
    );
  }

  const url = `amqp://${user}:${password}@${host}:${port}`;

  try {
    const connection = await amqp.connect(url);
    const channel = await connection.createChannel();
    return { connection, channel };
  } catch (error) {
    console.error('Không thể kết nối tới RabbitMQ:', error);
    throw new Error('Không thể kết nối tới RabbitMQ');
  }
};

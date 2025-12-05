import { connectRabbitMQ } from 'src/libs/rabbitmq.config';

export const publisher = async (data) => {
  const queue = 'payment_events';
  const dlxExchange = 'dlx.payment';
  const { connection, channel } = await connectRabbitMQ();

  try {
    // Declare the dead-letter exchange
    await channel.assertExchange(dlxExchange, 'topic', {
      durable: true,
    });

    // Assert queue with dead-letter exchange configuration
    await channel.assertQueue(queue, {
      durable: true,
      autoDelete: false,
      exclusive: false,
      arguments: {
        'x-dead-letter-exchange': dlxExchange,
        'x-dead-letter-routing-key': 'payment_events.failed',
      },
    });

    const payload = {
      ...data,
      amount: parseFloat(data.amount),
      items: data.items?.map((item) => ({
        ...item,
        price: parseFloat(item.price),
      })),
    };

    channel.sendToQueue(queue, Buffer.from(JSON.stringify(payload)), {
      persistent: true,
    });
  } finally {
    await channel.close();
    await connection.close();
  }
};

import { SendPayloadDto } from '@/libs/types/send-payload.dto';
import { PushNotiService } from '@/services/push-noti.service';
import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import * as amqp from 'amqplib';
import { connectRabbitMQ } from '../libs/rabbitmq.config';

@Injectable()
export class PushNotificationConsumer implements OnModuleInit {
  private readonly logger = new Logger(PushNotificationConsumer.name);
  private connection: amqp.Connection;
  private channel: amqp.Channel;
  private readonly queueName = 'push_noti_events';

  constructor(private readonly pushNotificationService: PushNotiService) {}

  async onModuleInit() {
    await this.connectToRabbitMQ();
    this.startConsuming();
  }

  private async connectToRabbitMQ() {
    try {
      const { connection, channel } = await connectRabbitMQ();
      this.connection = connection;
      this.channel = channel;
      await this.channel.assertQueue(this.queueName, { durable: true });
      this.logger.log(`Connected to RabbitMQ and asserted queue: ${this.queueName}`);
    } catch (error) {
      this.logger.error('Failed to connect to RabbitMQ', error);
      throw error;
    }
  }

  private startConsuming() {
    this.channel.consume(this.queueName, async msg => {
      if (msg) {
        try {
          const messageContent = msg.content.toString();
          const payload: SendPayloadDto = JSON.parse(messageContent);

          await this.pushNotificationService.send(payload);

          this.channel.ack(msg);
          this.logger.log('Notification sent successfully');
        } catch (error) {
          this.logger.error('Error processing message', error);
          this.channel.nack(msg, false, false);
        }
      }
    });
  }

  async onModuleDestroy() {
    if (this.channel) {
      await this.channel.close();
    }
    if (this.connection) {
      await this.connection.close();
    }
  }
}

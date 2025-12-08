import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { NotificationGateway } from './services/notification.gateway';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  const httpServer = app.getHttpServer();

  const notificationGateway = app.get(NotificationGateway);
  notificationGateway.initialize(httpServer);

  const port = process.env.PORT ?? 8091;
  await app.listen(port);
  console.log(`Application is running on: http://localhost:${port}`);
  console.log(`WebSocket is running on: ws://localhost:${port}/noti/public/ws`);
}
void bootstrap();

export default bootstrap;

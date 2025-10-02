import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  app.setGlobalPrefix('payment');
  console.log({
    PAYOS_EXPIRED_DURATION: process.env.PAYOS_EXPIRED_DURATION,
    PAYOS_CLIENT_ID: process.env.PAYOS_CLIENT_ID,
    PAYOS_API_KEY: process.env.PAYOS_API_KEY,
    PAYOS_CHECKSUM_KEY: process.env.PAYOS_CHECKSUM_KEY,
  });

  await app.listen(process.env.PORT ?? 3000);
}
void bootstrap();

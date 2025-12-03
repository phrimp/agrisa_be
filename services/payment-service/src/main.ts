import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { ResponseInterceptor } from './common/response.interceptor';
import { GlobalExceptionFilter } from './common/exception.filter';
import { winstonLoggerService } from './logger';

async function bootstrap() {
  process.env.TZ = 'Asia/Ho_Chi_Minh';
  const app = await NestFactory.create(AppModule, {
    logger: winstonLoggerService,
  });
  app.setGlobalPrefix('payment');
  app.useGlobalInterceptors(new ResponseInterceptor());
  app.useGlobalFilters(new GlobalExceptionFilter());
  await app.listen(process.env.PORT ?? 3000);
}
void bootstrap();

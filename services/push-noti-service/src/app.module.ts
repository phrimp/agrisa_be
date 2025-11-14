import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { NotiController } from './controllers/noti.controller';

@Module({
  imports: [],
  controllers: [AppController, NotiController],
  providers: [AppService],
})
export class AppModule {}

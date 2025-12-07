import { SendPayloadDto } from '@/libs/types/send-payload.dto';
import { SubscribeDto } from '@/libs/types/subscribe.dto';
import { PushNotiService } from '@/services/push-noti.service';
import { Body, Controller, Headers, Post } from '@nestjs/common';

@Controller('noti')
export class PushNotiController {
  constructor(private readonly pushNotiService: PushNotiService) {}

  @Post('private/subscribe/web')
  async subscribe(@Headers('x-user-id') userId: string, @Body() data: SubscribeDto) {
    return this.pushNotiService.subscribeWeb(userId, data);
  }

  @Post('private/subscribe/android')
  async subscribeAndroid(@Headers('x-user-id') userId: string, @Body() data: SubscribeDto) {
    return this.pushNotiService.subscribeAndroid(userId, data);
  }

  @Post('private/subscribe/ios')
  async subscribeIOS(@Headers('x-user-id') userId: string, @Body() data: SubscribeDto) {
    return this.pushNotiService.subscribeIOS(userId, data);
  }

  @Post('private/unsubscribe/web')
  async unsubscribeWeb(@Headers('x-user-id') userId: string) {
    return this.pushNotiService.unsubscribeWeb(userId);
  }

  @Post('private/unsubscribe/android')
  async unsubscribeAndroid(@Headers('x-user-id') userId: string) {
    return this.pushNotiService.unsubscribeAndroid(userId);
  }

  @Post('private/unsubscribe/ios')
  async unsubscribeIOS(@Headers('x-user-id') userId: string) {
    return this.pushNotiService.unsubscribeIOS(userId);
  }

  @Post('public/send/web')
  async sendWeb(@Body() data: SendPayloadDto) {
    return this.pushNotiService.sendWeb(data);
  }

  @Post('public/send/android')
  async sendAndroid(@Body() data: SendPayloadDto) {
    return this.pushNotiService.sendAndroid(data);
  }

  @Post('public/send/ios')
  async sendIOS(@Body() data: SendPayloadDto) {
    return this.pushNotiService.sendIOS(data);
  }

  @Post('public/send')
  async send(@Body() data: SendPayloadDto) {
    return this.pushNotiService.send(data);
  }
}

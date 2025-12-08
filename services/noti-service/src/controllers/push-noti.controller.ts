import { SendPayloadDto } from '@/libs/types/send-payload.dto';
import { SubscribeDto } from '@/libs/types/subscribe.dto';
import { PushNotiService } from '@/services/push-noti.service';
import { Body, Controller, Get, Headers, Post, Query } from '@nestjs/common';

@Controller('noti')
export class PushNotiController {
  constructor(private readonly pushNotiService: PushNotiService) {}

  @Post('protected/subscribe/web')
  async subscribe(@Headers('x-user-id') userId: string, @Body() data: SubscribeDto) {
    return this.pushNotiService.subscribeWeb(userId, data);
  }

  @Post('protected/subscribe/android')
  async subscribeAndroid(@Headers('x-user-id') userId: string, @Body() data: SubscribeDto) {
    return this.pushNotiService.subscribeAndroid(userId, data);
  }

  @Post('protected/subscribe/ios')
  async subscribeIOS(@Headers('x-user-id') userId: string) {
    return this.pushNotiService.subscribeIOS(userId);
  }

  @Post('protected/unsubscribe/web')
  async unsubscribeWeb(@Headers('x-user-id') userId: string) {
    return this.pushNotiService.unsubscribeWeb(userId);
  }

  @Post('protected/unsubscribe/android')
  async unsubscribeAndroid(@Headers('x-user-id') userId: string) {
    return this.pushNotiService.unsubscribeAndroid(userId);
  }

  @Post('protected/unsubscribe/ios')
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

  @Get('protected/validate')
  async me(@Headers('x-user-id') userId: string, @Query('platform') platform: string) {
    return await this.pushNotiService.isSubcribed(userId, platform);
  }

  @Get('public/pull/ios')
  async pullNotifications(@Query('user_id') userId: string, @Query('limit') limit?: number) {
    return await this.pushNotiService.pullNotifications(userId, limit ? +limit : 10);
  }

  @Post('protected/mark-read')
  async markAsRead(@Body('receiverIds') receiverIds: string[]) {
    return await this.pushNotiService.markAsRead(receiverIds);
  }

  @Get('protected/notifications')
  async getAllNotifications(
    @Headers('x-user-id') userId: string,
    @Query('page') page?: number,
    @Query('limit') limit?: number,
    @Query('status') status?: string,
  ) {
    return await this.pushNotiService.getAllNotifications(
      userId,
      page ? +page : 1,
      limit ? +limit : 20,
      status,
    );
  }
}

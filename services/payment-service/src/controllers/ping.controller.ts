import { Controller, Get, Inject } from '@nestjs/common';
import type { PingService } from '../services/ping.service';

@Controller('payment')
export class PingController {
  constructor(
    @Inject('PingService') private readonly pingService: PingService,
  ) {}

  @Get('/public/ping')
  publicPing() {
    return this.pingService.ping();
  }

  @Get('/private/ping')
  privatePing() {
    return this.pingService.ping();
  }
}

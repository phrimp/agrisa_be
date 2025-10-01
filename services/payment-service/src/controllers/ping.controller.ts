import { Controller, Get, Inject } from '@nestjs/common';
import type { PingService } from '../services/ping.service';

@Controller()
export class PingController {
  constructor(
    @Inject('PingService') private readonly pingService: PingService,
  ) {}

  @Get('/public/ping')
  publicPing() {
    return this.pingService.ping();
  }

  @Get('/protected/ping')
  protectedPing() {
    return this.pingService.ping();
  }
}

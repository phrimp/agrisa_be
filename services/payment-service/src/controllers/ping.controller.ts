import { Controller, Get, Inject } from '@nestjs/common';
import type { PingService } from '../services/ping.service';

@Controller('payment')
export class PingController {
  constructor(
    @Inject('PingService') private readonly pingService: PingService,
  ) {}

  @Get('ping')
  ping() {
    return this.pingService.ping();
  }
}

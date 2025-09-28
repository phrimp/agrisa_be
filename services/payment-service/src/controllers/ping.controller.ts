import { Controller, Get, Inject } from '@nestjs/common';
import type { PingService } from '../services/ping.service';

@Controller('ping')
export class PingController {
  constructor(
    @Inject('PingService') private readonly pingService: PingService,
  ) {}

  @Get()
  ping() {
    return this.pingService.ping();
  }
}

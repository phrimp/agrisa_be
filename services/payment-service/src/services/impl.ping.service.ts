import { Injectable } from '@nestjs/common';
import { PingResponse } from '../entities/ping.entity';
import { PingService } from './ping.service';

@Injectable()
export class ImplPingService implements PingService {
  ping(): PingResponse {
    return {
      message: 'pong',
    };
  }
}

import { Injectable } from '@nestjs/common';
import { PingResponse } from '../types/ping.types';
import { PingService } from './ping.service';

@Injectable()
export class ImplPingService implements PingService {
  ping(): PingResponse {
    return {
      message: 'pong',
    };
  }
}

import { PingResponse } from '../entities/ping.entity';

export interface PingService {
  ping(): PingResponse;
}

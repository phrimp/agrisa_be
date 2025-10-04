import { PingResponse } from '../types/ping.types';

export interface PingService {
  ping(): PingResponse;
}

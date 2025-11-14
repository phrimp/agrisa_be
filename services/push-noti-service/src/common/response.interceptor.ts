/* eslint-disable @typescript-eslint/no-unsafe-assignment */
import {
  Injectable,
  NestInterceptor,
  ExecutionContext,
  CallHandler,
} from '@nestjs/common';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

@Injectable()
export class ResponseInterceptor implements NestInterceptor {
  intercept(context: ExecutionContext, next: CallHandler): Observable<any> {
    return next.handle().pipe(
      map((data) => {
        if (data && typeof data === 'object' && 'metadata' in data) {
          // If response already has metadata, merge it with timestamp
          const { metadata, ...rest } = data;
          return {
            success: true,
            data: rest,
            metadata: {
              ...metadata,
              timestamp: new Date().toISOString(),
            },
          };
        } else {
          // Normal wrapping
          return {
            success: true,
            data: data ?? {},
            meta: {
              timestamp: new Date().toISOString(),
            },
          };
        }
      }),
    );
  }
}

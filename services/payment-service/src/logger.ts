import * as winston from 'winston';
import DailyRotateFile from 'winston-daily-rotate-file';
import { LoggerService } from '@nestjs/common';

const customLevels = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  debug: 4,
  verbose: 5,
};

const winstonLogger = winston.createLogger({
  level: 'info',
  levels: customLevels,
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.errors({ stack: true }),
    winston.format.json(),
  ),
  transports: [
    new winston.transports.Console(),
    new DailyRotateFile({
      filename: 'log_%DATE%.log',
      dirname: '/app/log',
      datePattern: 'YYYY-MM-DD',
      maxSize: '20m',
      maxFiles: '14d',
    }),
  ],
});

export class WinstonLoggerService implements LoggerService {
  log(message: any, context?: string) {
    winstonLogger.info(String(message), { context });
  }

  error(message: any, stack?: string, context?: string) {
    winstonLogger.error(String(message), { stack, context });
  }

  warn(message: any, context?: string) {
    winstonLogger.warn(String(message), { context });
  }

  debug(message: any, context?: string) {
    winstonLogger.debug(String(message), { context });
  }

  verbose(message: any, context?: string) {
    winstonLogger.verbose(String(message), { context });
  }
}

export const winstonLoggerService = new WinstonLoggerService();

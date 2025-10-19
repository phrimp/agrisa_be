import * as winston from 'winston';
import DailyRotateFile from 'winston-daily-rotate-file';

export const winstonLogger = winston.createLogger({
  level: 'info',
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

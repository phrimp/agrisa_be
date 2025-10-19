import * as winston from 'winston';
import DailyRotateFile from 'winston-daily-rotate-file';

// Define custom levels to match NestJS logger levels
const customLevels = {
  error: 0,
  warn: 1,
  log: 2,
  info: 3,
  debug: 4,
  verbose: 5,
};

export const winstonLogger = winston.createLogger({
  level: 'log',
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

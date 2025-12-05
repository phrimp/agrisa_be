import { TypeOrmModuleOptions } from '@nestjs/typeorm';

export const databaseConfig: TypeOrmModuleOptions = {
  type: 'postgres',
  host: process.env.POSTGRES_HOST || 'localhost',
  port: process.env.POSTGRES_PORT ? Number(process.env.POSTGRES_PORT) : 5432,
  username: process.env.POSTGRES_USER || 'postgres',
  password: process.env.POSTGRES_PASSWORD || '123456',
  database: 'push_noti_service',
  entities: [__dirname + '/../**/*.entity{.ts,.js}'],
  synchronize: process.env.BUN_ENV !== 'production',
  extra: {
    timezone: 'Asia/Ho_Chi_Minh',
  },
};

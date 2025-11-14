import { Controller, Get, Header } from '@nestjs/common';
import { readFile } from 'fs/promises';
import * as path from 'path';

@Controller('pushed-noti')
export class NotiController {
  @Get('permission')
  @Header('Content-Type', 'text/html')
  async getPermission() {
    const filePath = path.join(
      process.cwd(),
      'src',
      'libs',
      'permission',
      'index.html',
    );
    return readFile(filePath, 'utf-8');
  }
}

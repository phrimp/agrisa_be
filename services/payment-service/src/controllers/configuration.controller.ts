import { Body, Controller, Get, Inject, Patch } from '@nestjs/common';
import { Configuration } from '../entities/configuration.entity';
import type { ConfigurationService } from '../services/configuration.service';

@Controller('configuration')
export class ConfigurationController {
  constructor(
    @Inject('ConfigurationService')
    private readonly configurationService: ConfigurationService,
  ) {}

  @Get()
  async getConfiguration() {
    return this.configurationService.getConfiguration();
  }

  @Patch()
  async updateConfiguration(@Body() body: Configuration) {
    return this.configurationService.updateConfiguration(body);
  }
}

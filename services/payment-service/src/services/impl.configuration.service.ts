import { Injectable } from '@nestjs/common';
import { Configuration } from 'src/entities/configuration.entity';
import { ConfigurationRepository } from 'src/repositories/configuration.repository';
import { ConfigurationService } from './configuration.service';

@Injectable()
export class ImplConfigurationService implements ConfigurationService {
  constructor(
    private readonly configurationRepository: ConfigurationRepository,
  ) {}
  async getConfiguration(): Promise<Configuration | null> {
    return this.configurationRepository.getConfiguration();
  }
  async updateConfiguration(
    updates: Partial<Configuration>,
  ): Promise<Configuration | null> {
    const config = await this.configurationRepository.getConfiguration();
    if (!config) return null;

    const updatedConfig = { ...config, ...updates };
    await this.configurationRepository.updateConfiguration(updatedConfig);
    return updatedConfig;
  }
}

import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Configuration } from 'src/entities/configuration.entity';
import { ConfigurationService } from './configuration.service';

@Injectable()
export class ImplConfigurationService implements ConfigurationService {
  constructor(
    @InjectRepository(Configuration)
    private readonly configRepo: Repository<Configuration>,
  ) {}

  async getConfiguration(): Promise<Configuration | null> {
    return this.configRepo.findOne({ where: {} });
  }

  async updateConfiguration(
    updates: Partial<Configuration>,
  ): Promise<Configuration | null> {
    const config = await this.getConfiguration();
    if (!config) return null;

    const updatedConfig = { ...config, ...updates };
    await this.configRepo.save(updatedConfig);
    return updatedConfig;
  }
}

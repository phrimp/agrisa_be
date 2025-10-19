import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Configuration } from '../entities/configuration.entity';

@Injectable()
export class ConfigurationRepository {
  constructor(
    @InjectRepository(Configuration)
    private readonly configRepo: Repository<Configuration>,
  ) {}

  async getConfiguration(): Promise<Configuration | null> {
    return this.configRepo.findOne({ where: {} });
  }

  async updateConfiguration(
    configuration: Configuration,
  ): Promise<Configuration | null> {
    const config = await this.getConfiguration();
    if (!config) {
      return null;
    }
    return this.configRepo.save({ ...config, ...configuration });
  }
}

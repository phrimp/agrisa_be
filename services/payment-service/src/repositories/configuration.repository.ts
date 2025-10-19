import { Configuration } from './../entities/configuration.entity';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';

@Injectable()
export class ConfigurationRepository {
  constructor(
    @InjectRepository(Configuration)
    private readonly configurationRepository: Repository<Configuration>,
  ) {}
  async getConfiguration(): Promise<Configuration | null> {
    return await this.configurationRepository.findOne({});
  }

  async updateConfiguration(
    configuration: Configuration,
  ): Promise<Configuration | null> {
    const config = await this.getConfiguration();
    if (!config) {
      return null;
    }
    return this.configurationRepository.save({ ...config, ...configuration });
  }
}

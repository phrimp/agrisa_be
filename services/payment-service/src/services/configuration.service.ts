import { Configuration } from './../entities/configuration.entity';
export interface ConfigurationService {
  getConfiguration(): Promise<Configuration | null>;
  updateConfiguration(
    updates: Partial<Configuration>,
  ): Promise<Configuration | null>;
}

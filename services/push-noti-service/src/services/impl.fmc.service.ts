import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Fmc } from 'src/entities/fmc.entity';

@Injectable()
export class ImplFmcService {
  constructor(
    @InjectRepository(Fmc)
    private readonly fmcRepo: Repository<Fmc>,
  ) {}

  async registerFmc({
    fmc_token,
    user_id,
  }: {
    fmc_token: string;
    user_id: string;
  }) {
    const data = this.fmcRepo.create({
      fmc_token,
      user_id,
      created_at: new Date(),
      updated_at: new Date(),
    });
    return await this.fmcRepo.save(data);
  }

  async updateFmc({
    fmc_token,
    user_id,
  }: {
    fmc_token: string;
    user_id: string;
  }) {
    const existtingFmc = await this.fmcRepo.findOne({ where: { user_id } });
    if (!existtingFmc) {
      throw new Error('Chưa đăng ký fmc token cho thiết bị này');
    }
    existtingFmc.fmc_token = fmc_token;
    existtingFmc.updated_at = new Date();
    return await this.fmcRepo.save(existtingFmc);
  }
}

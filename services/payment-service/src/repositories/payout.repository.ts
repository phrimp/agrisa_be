import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Payout } from '../entities/payout.entity';

@Injectable()
export class PayoutRepository {
  constructor(
    @InjectRepository(Payout)
    private readonly payoutRepo: Repository<Payout>,
  ) {}

  async create(payout: Partial<Payout>): Promise<Payout> {
    const newPayout = this.payoutRepo.create(payout);
    return await this.payoutRepo.save(newPayout);
  }

  async findById(id: string): Promise<Payout | null> {
    return await this.payoutRepo.findOne({
      where: { id },
    });
  }

  async update(id: string, updates: Partial<Payout>): Promise<Payout | null> {
    await this.payoutRepo.update(id, updates);
    return await this.findById(id);
  }

  async findByUserId(
    user_id: string,
    page: number,
    limit: number,
  ): Promise<{ items: Payout[]; total: number }> {
    const page_num = Math.max(1, Number(page) || 1);
    const limit_num = Math.max(1, Number(limit) || 10);
    const skip = (page_num - 1) * limit_num;

    const [items, total] = await this.payoutRepo.findAndCount({
      where: { user_id, status: 'completed' },
      skip,
      take: limit_num,
      order: { created_at: 'DESC' },
    });
    return { items, total };
  }

  async findByIdAndUserId(id: string, user_id: string): Promise<Payout | null> {
    return await this.payoutRepo.findOne({
      where: { id, user_id },
    });
  }
}

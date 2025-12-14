import { Injectable } from '@nestjs/common';
import { Payout } from '../entities/payout.entity';
import { PayoutRepository } from '../repositories/payout.repository';
import { PayoutService } from './payout.service';

@Injectable()
export class ImplPayoutService implements PayoutService {
  constructor(private readonly payoutRepository: PayoutRepository) {}
  // getTotalPayoutAmountByTypeAndUserId(type: string, user_id: string): Promise<number> {
  //   throw new Error('Method not implemented.');
  // }

  async create(payout: Partial<Payout>): Promise<Payout> {
    return this.payoutRepository.create(payout);
  }

  async findById(id: string): Promise<Payout | null> {
    return this.payoutRepository.findById(id);
  }

  async update(id: string, updates: Partial<Payout>): Promise<Payout | null> {
    return this.payoutRepository.update(id, updates);
  }

  async findByUserId(
    user_id: string,
    page: number,
    limit: number,
  ): Promise<{ items: Payout[]; total: number }> {
    return this.payoutRepository.findByUserId(user_id, page, limit);
  }

  async findByIdAndUserId(id: string, user_id: string): Promise<Payout | null> {
    return this.payoutRepository.findByIdAndUserId(id, user_id);
  }

  async findByItemIds(item_ids: string[]): Promise<Payout[]> {
    return this.payoutRepository.findByItemIds(item_ids);
  }

  // async getTotalPayoutAmountByTypeAndUserId(
  //   type: string,
  //   user_id: string,
  // ): Promise<number> {
  //   return this.payoutRepository.getTotalPayoutAmountByTypeAndUserId(
  //     type,
  //     user_id,
  //   );
  // }
}

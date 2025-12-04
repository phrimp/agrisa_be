import { Payout } from '../entities/payout.entity';

export interface PayoutService {
  create(payout: Partial<Payout>): Promise<Payout>;
  findById(id: string): Promise<Payout | null>;
  update(id: string, updates: Partial<Payout>): Promise<Payout | null>;
  findByUserId(
    user_id: string,
    page: number,
    limit: number,
  ): Promise<{ items: Payout[]; total: number }>;
  findByIdAndUserId(id: string, user_id: string): Promise<Payout | null>;
}

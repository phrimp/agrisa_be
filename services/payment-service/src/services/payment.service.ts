import { Payment } from '../entities/payment.entity';

export interface PaymentService {
  create(payment: Partial<Payment>): Promise<Payment>;
  find(): Promise<Payment[]>;
  findById(id: string): Promise<Payment | null>;
  update(id: string, updates: Partial<Payment>): Promise<Payment | null>;
  delete(id: string): Promise<boolean>;
  findByUserId(
    user_id: string,
    page: number,
    limit: number,
    status?: string[],
  ): Promise<Payment[]>;
}

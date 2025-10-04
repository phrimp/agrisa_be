import { Payment } from '../entities/payment.entity';

export interface PaymentService {
  create(payment: Partial<Payment>): Promise<Payment>;
  find(
    page: number,
    limit: number,
    status?: string[],
  ): Promise<{ items: Payment[]; total: number }>;
  findById(id: string): Promise<Payment | null>;
  findByOrderCode(order_code: string): Promise<Payment | null>;
  update(id: string, updates: Partial<Payment>): Promise<Payment | null>;
  delete(id: string): Promise<boolean>;
  findByUserId(
    user_id: string,
    page: number,
    limit: number,
    status?: string[],
  ): Promise<{ items: Payment[]; total: number }>;
}

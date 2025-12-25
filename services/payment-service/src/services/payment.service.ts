import { Payment } from '../entities/payment.entity';

export interface PaymentService {
  create(payment: Partial<Payment>): Promise<Payment>;
  find(
    page: number,
    limit: number,
    status?: string[],
  ): Promise<{ items: Payment[]; total: number }>;
  findById(id: string): Promise<Payment | null>;
  findByIdAndUserId(id: string, user_id: string): Promise<Payment | null>;
  findByOrderCode(order_code: string): Promise<Payment | null>;
  update(id: string, updates: Partial<Payment>): Promise<Payment | null>;
  delete(id: string): Promise<boolean>;
  findByUserId(
    user_id: string,
    page: number,
    limit: number,
    status?: string[],
  ): Promise<{ items: Payment[]; total: number }>;
  findExpired(): Promise<Payment[]>;
  getTotalAmountByUserAndType(user_id: string, type: string): Promise<number>;
  getTotalAmountByType(type: string): Promise<number>;
  getAllOrdersAdmin(): Promise<Payment[]>;
}

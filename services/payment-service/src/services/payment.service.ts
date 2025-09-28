import { Payment } from '../entities/payment.entity';

export interface PaymentService {
  create(payment: Partial<Payment>): Promise<Payment>;
  getAll(): Promise<Payment[]>;
  getById(id: string): Promise<Payment | null>;
  update(id: string, updates: Partial<Payment>): Promise<Payment | null>;
  delete(id: string): Promise<boolean>;
}

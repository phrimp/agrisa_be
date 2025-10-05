import { Injectable } from '@nestjs/common';
import { PaymentRepository } from '../repositories/payment.repository';
import { Payment } from '../entities/payment.entity';
import { PaymentService } from './payment.service';

@Injectable()
export class ImplPaymentService implements PaymentService {
  constructor(private readonly paymentRepository: PaymentRepository) {}

  async create(payment: Partial<Payment>): Promise<Payment> {
    return this.paymentRepository.create(payment);
  }

  async find(
    page: number,
    limit: number,
    status?: string[],
  ): Promise<{ items: Payment[]; total: number }> {
    return this.paymentRepository.find(page, limit, status ?? []);
  }

  async findById(id: string): Promise<Payment | null> {
    return this.paymentRepository.findById(id);
  }

  async findByOrderCode(order_code: string): Promise<Payment | null> {
    return this.paymentRepository.findByOrderCode(order_code);
  }

  async update(id: string, updates: Partial<Payment>): Promise<Payment | null> {
    return this.paymentRepository.update(id, updates);
  }

  async delete(id: string): Promise<boolean> {
    return this.paymentRepository.delete(id);
  }

  async findByUserId(
    user_id: string,
    page: number,
    limit: number,
    status?: string[],
  ): Promise<{ items: Payment[]; total: number }> {
    return this.paymentRepository.findByUserId(
      user_id,
      page,
      limit,
      status ?? [],
    );
  }

  async findExpired(): Promise<Payment[]> {
    return this.paymentRepository.findExpired();
  }
}

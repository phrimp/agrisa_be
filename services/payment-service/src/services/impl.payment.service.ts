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

  async find(): Promise<Payment[]> {
    return this.paymentRepository.findAll();
  }

  async findById(id: string): Promise<Payment | null> {
    return this.paymentRepository.findById(id);
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
  ): Promise<Payment[]> {
    return this.paymentRepository.findByUserId(
      user_id,
      page,
      limit,
      status ?? [],
    );
  }
}

import { Injectable } from '@nestjs/common';
import { Payment } from '../entities/payment.entity';
import { PaymentRepository } from '../repositories/payment.repository';
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

  async findByIdAndUserId(
    id: string,
    user_id: string,
  ): Promise<Payment | null> {
    return this.paymentRepository.findByIdAndUserId(id, user_id);
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

  async getTotalAmountByUserAndType(
    user_id: string,
    type: string,
  ): Promise<number> {
    return this.paymentRepository.getTotalAmountByUserAndType(user_id, type);
  }

  async getTotalAmountByType(type: string): Promise<number> {
    return this.paymentRepository.getTotalAmountByType(type);
  }

  async getTotalAmountByTypeAndDateRange(
    type: string,
    from?: Date,
    to?: Date,
  ): Promise<number> {
    return this.paymentRepository.getTotalAmountByTypeAndDateRange(
      type,
      from,
      to,
    );
  }

  async getAllOrdersAdmin() {
    return this.paymentRepository.getAllOrdersAdmin();
  }

  async getOrderByIdAdmin(id: string) {
    return this.paymentRepository.findById(id);
  }
}

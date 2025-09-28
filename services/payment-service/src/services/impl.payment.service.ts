import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Payment } from '../entities/payment.entity';
import { PaymentService } from './payment.service';

@Injectable()
export class ImplPaymentService implements PaymentService {
  constructor(
    @InjectRepository(Payment)
    private paymentRepository: Repository<Payment>,
  ) {}

  async create(payment: Partial<Payment>): Promise<Payment> {
    const newPayment = this.paymentRepository.create(payment);
    return this.paymentRepository.save(newPayment);
  }

  async getAll(): Promise<Payment[]> {
    return this.paymentRepository.find();
  }

  async getById(id: string): Promise<Payment | null> {
    return this.paymentRepository.findOne({ where: { id } });
  }

  async update(id: string, updates: Partial<Payment>): Promise<Payment | null> {
    await this.paymentRepository.update(id, updates);
    return this.getById(id);
  }

  async delete(id: string): Promise<boolean> {
    const result = await this.paymentRepository.delete(id);
    return typeof result.affected === 'number' && result.affected > 0;
  }
}

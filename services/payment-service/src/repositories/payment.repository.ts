import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { In, Repository } from 'typeorm';
import { Payment } from '../entities/payment.entity';

@Injectable()
export class PaymentRepository {
  constructor(
    @InjectRepository(Payment)
    private readonly paymentRepo: Repository<Payment>,
  ) {}

  async create(payment: Partial<Payment>): Promise<Payment> {
    const newPayment = this.paymentRepo.create(payment);
    return this.paymentRepo.save(newPayment);
  }

  async findAll(): Promise<Payment[]> {
    return this.paymentRepo.find();
  }

  async findById(id: string): Promise<Payment | null> {
    return this.paymentRepo.findOne({ where: { id } });
  }

  async update(id: string, updates: Partial<Payment>): Promise<Payment | null> {
    await this.paymentRepo.update(id, updates);
    return this.findById(id);
  }

  async delete(id: string): Promise<boolean> {
    const result = await this.paymentRepo.delete(id);
    return typeof result.affected === 'number' && result.affected > 0;
  }

  async findByUserId(
    user_id: string,
    page: number,
    limit: number,
    status: string[],
  ): Promise<Payment[]> {
    const page_num = Math.max(1, Number(page) || 1);
    const limit_num = Math.max(1, Number(limit) || 10);
    const skip = (page_num - 1) * limit_num;

    const query: Record<string, unknown> = { user_id };
    if (status && status.length > 0) {
      query.status = In(status);
    }
    return this.paymentRepo.find({
      where: query,
      skip,
      take: limit_num,
      order: { created_at: 'DESC' },
    });
  }
}

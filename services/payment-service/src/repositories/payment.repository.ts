import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { In, LessThan, Not, Repository, type FindManyOptions } from 'typeorm';
import { Payment } from '../entities/payment.entity';

@Injectable()
export class PaymentRepository {
  constructor(
    @InjectRepository(Payment)
    private readonly paymentRepo: Repository<Payment>,
  ) {}

  async create(payment: Partial<Payment>): Promise<Payment> {
    const newPayment = this.paymentRepo.create(payment);
    return await this.paymentRepo.save(newPayment);
  }

  async find(
    page: number,
    limit: number,
    status: string[],
  ): Promise<{ items: Payment[]; total: number }> {
    const page_num = Math.max(1, Number(page) || 1);
    const limit_num = Math.max(1, Number(limit) || 10);
    const skip = (page_num - 1) * limit_num;

    const options: FindManyOptions<Payment> = {
      skip,
      take: limit_num,
      order: { created_at: 'DESC' },
      relations: ['orderItems'],
    };

    if (status && status.length > 0) {
      options.where = { status: In(status) };
    }

    const [items, total] = await this.paymentRepo.findAndCount(options);

    return { items, total };
  }

  async findById(id: string): Promise<Payment | null> {
    return await this.paymentRepo.findOne({
      where: { id },
      relations: ['orderItems'],
    });
  }

  async findByIdAndUserId(
    id: string,
    user_id: string,
  ): Promise<Payment | null> {
    return await this.paymentRepo.findOne({
      where: { id, user_id },
      relations: ['orderItems'],
    });
  }

  async findByOrderCode(order_code: string): Promise<Payment | null> {
    return await this.paymentRepo.findOne({
      where: { order_code },
      relations: ['orderItems'],
    });
  }

  async update(id: string, updates: Partial<Payment>): Promise<Payment | null> {
    await this.paymentRepo.update(id, updates);
    return await this.findById(id);
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
  ): Promise<{ items: Payment[]; total: number }> {
    const page_num = Math.max(1, Number(page) || 1);
    const limit_num = Math.max(1, Number(limit) || 10);
    const skip = (page_num - 1) * limit_num;

    const options: FindManyOptions<Payment> = {
      where: { user_id },
      skip,
      take: limit_num,
      order: { created_at: 'DESC' },
      relations: ['orderItems'],
    };

    if (status && status.length > 0) {
      options.where = { user_id, status: In(status) };
    }

    const [items, total] = await this.paymentRepo.findAndCount(options);
    return { items, total };
  }

  async findExpired(): Promise<Payment[]> {
    return await this.paymentRepo.find({
      where: {
        expired_at: LessThan(new Date()),
        status: Not(In(['completed', 'cancelled', 'expired'])),
      },
    });
  }
}

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
      relations: ['items'],
    };

    if (status && status.length > 0) {
      options.where = { status: In(status) };
    }

    const [items, total] = await this.paymentRepo.findAndCount(options);

    return { items, total };
  }

  async findById(id: string) {
    return await this.paymentRepo.findOne({
      where: { id },
      relations: ['items'],
    });
  }

  async findByIdAndUserId(
    id: string,
    user_id: string,
  ): Promise<Payment | null> {
    return await this.paymentRepo.findOne({
      where: { id, user_id },
      relations: ['items'],
    });
  }

  async findByOrderCode(order_code: string): Promise<Payment | null> {
    return await this.paymentRepo.findOne({
      where: { order_code },
      relations: ['items'],
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
      relations: ['items'],
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

  async getTotalAmountByUserAndType(user_id: string, type: string) {
    const result = await this.paymentRepo
      .createQueryBuilder('payment')
      .select('SUM(payment.amount)', 'total')
      .where('payment.user_id = :user_id', { user_id })
      .andWhere('payment.type = :type', { type })
      .andWhere('payment.status = :status', { status: 'completed' })
      .getRawOne();
    return Number(result?.total) || 0;
  }

  async getTotalPayoutByUserAndType(
    user_id: string,
    type: string,
  ): Promise<number> {
    const result = await this.paymentRepo
      .createQueryBuilder('payment')
      .leftJoin('payment.payouts', 'payout')
      .select('SUM(payout.amount)', 'total')
      .where('payment.type = :type', { type })
      .andWhere('payment.status = :status', { status: 'completed' })
      .andWhere('payout.user_id = :user_id', { user_id })
      .getRawOne();
    return Number(result?.total) || 0;
  }

  async getTotalAmountByType(type: string): Promise<number> {
    const result = await this.paymentRepo
      .createQueryBuilder('payment')
      .select('SUM(payment.amount)', 'total')
      .where('payment.type = :type', { type })
      .andWhere('payment.status = :status', { status: 'completed' })
      .getRawOne();
    return Number(result?.total) || 0;
  }

  async getTotalAmountByTypeAndDateRange(
    type: string,
    from?: Date,
    to?: Date,
  ): Promise<number> {
    const query = this.paymentRepo
      .createQueryBuilder('payment')
      .select('SUM(payment.amount)', 'total')
      .where('payment.type = :type', { type })
      .andWhere('payment.status = :status', { status: 'completed' });

    if (from) {
      query.andWhere('payment.created_at >= :from', { from });
    }
    if (to) {
      query.andWhere('payment.created_at <= :to', { to });
    }

    const result = await query.getRawOne();
    return Number(result?.total) || 0;
  }

  async getAllOrdersAdmin() {
    return await this.paymentRepo.find({
      relations: {
        items: true,
      },
      order: { created_at: 'DESC' },
    });
  }
}

import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { OrderItem } from '../entities/order-item.entity';
import { Repository } from 'typeorm';

@Injectable()
export class OrderItemRepository {
  constructor(
    @InjectRepository(OrderItem)
    private readonly orderItemRepository: Repository<OrderItem>,
  ) {}

  async create(orderItem: Partial<OrderItem>): Promise<OrderItem> {
    const newOrderItem = this.orderItemRepository.create(orderItem);
    return await this.orderItemRepository.save(newOrderItem);
  }

  async findByPaymentId(payment_id: string): Promise<OrderItem[]> {
    return await this.orderItemRepository.find({ where: { payment_id } });
  }

  async deleteByPaymentId(payment_id: string): Promise<boolean> {
    const result = await this.orderItemRepository.delete({ payment_id });
    return typeof result.affected === 'number' && result.affected > 0;
  }

  async deleteById(id: string): Promise<boolean> {
    const result = await this.orderItemRepository.delete(id);
    return typeof result.affected === 'number' && result.affected > 0;
  }

  async findById(id: string): Promise<OrderItem | null> {
    return await this.orderItemRepository.findOne({ where: { id } });
  }

  async update(
    id: string,
    updates: Partial<OrderItem>,
  ): Promise<OrderItem | null> {
    await this.orderItemRepository.update(id, updates);
    return await this.findById(id);
  }
}

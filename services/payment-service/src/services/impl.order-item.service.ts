import { OrderItemService } from './order-item.service';
import { OrderItem } from 'src/entities/order-item.entity';
import { Injectable } from '@nestjs/common';
import { OrderItemRepository } from 'src/repositories/order-item.repository';

@Injectable()
export class ImplOrderItemService implements OrderItemService {
  constructor(private readonly orderItemRepository: OrderItemRepository) {}

  async create(orderItem: Partial<OrderItem>): Promise<OrderItem> {
    return await this.orderItemRepository.create(orderItem);
  }

  async findByPaymentId(payment_id: string): Promise<OrderItem[]> {
    return await this.orderItemRepository.findByPaymentId(payment_id);
  }

  async deleteByPaymentId(payment_id: string): Promise<boolean> {
    return await this.orderItemRepository.deleteByPaymentId(payment_id);
  }

  async deleteById(id: string): Promise<boolean> {
    return await this.orderItemRepository.deleteById(id);
  }

  async findById(id: string): Promise<OrderItem | null> {
    return await this.orderItemRepository.findById(id);
  }

  async update(
    id: string,
    updates: Partial<OrderItem>,
  ): Promise<OrderItem | null> {
    return await this.orderItemRepository.update(id, updates);
  }
}

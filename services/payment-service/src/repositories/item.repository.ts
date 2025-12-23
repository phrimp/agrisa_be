import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Item } from '../entities/item.entity';

@Injectable()
export class ItemRepository {
  constructor(
    @InjectRepository(Item)
    private readonly orderItemRepository: Repository<Item>,
  ) {}

  async create(orderItem: Partial<Item>): Promise<Item> {
    const newOrderItem = this.orderItemRepository.create(orderItem);
    return await this.orderItemRepository.save(newOrderItem);
  }

  async findByPaymentId(payment_id: string): Promise<Item[]> {
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

  async findById(id: string): Promise<Item | null> {
    return await this.orderItemRepository.findOne({ where: { id } });
  }

  async update(id: string, updates: Partial<Item>): Promise<Item | null> {
    await this.orderItemRepository.update(id, updates);
    return await this.findById(id);
  }

  async findByItemId(item_id: string): Promise<Item | null> {
    return await this.orderItemRepository.findOne({ where: { item_id } });
  }
}

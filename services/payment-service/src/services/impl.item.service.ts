import { Injectable } from '@nestjs/common';
import { Item } from '../entities/item.entity';
import { ItemRepository } from '../repositories/item.repository';
import { ItemService } from './item.service';

@Injectable()
export class ImplItemService implements ItemService {
  constructor(private readonly itemRepository: ItemRepository) {}

  async create(item: Partial<Item>): Promise<Item> {
    return await this.itemRepository.create(item);
  }

  async findByPaymentId(payment_id: string): Promise<Item[]> {
    return await this.itemRepository.findByPaymentId(payment_id);
  }

  async deleteByPaymentId(payment_id: string): Promise<boolean> {
    return await this.itemRepository.deleteByPaymentId(payment_id);
  }

  async deleteById(id: string): Promise<boolean> {
    return await this.itemRepository.deleteById(id);
  }

  async findById(id: string): Promise<Item | null> {
    return await this.itemRepository.findById(id);
  }

  async update(id: string, updates: Partial<Item>): Promise<Item | null> {
    return await this.itemRepository.update(id, updates);
  }
}

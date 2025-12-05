import { Item } from 'src/entities/item.entity';

export interface ItemService {
  create(orderItem: Partial<Item>): Promise<Item>;
  findByPaymentId(payment_id: string): Promise<Item[]>;
  deleteByPaymentId(payment_id: string): Promise<boolean>;
  deleteById(id: string): Promise<boolean>;
  findById(id: string): Promise<Item | null>;
  update(id: string, updates: Partial<Item>): Promise<Item | null>;
}

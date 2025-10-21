import { OrderItem } from 'src/entities/order-item.entity';

export interface OrderItemService {
  create(orderItem: Partial<OrderItem>): Promise<OrderItem>;
  findByPaymentId(payment_id: string): Promise<OrderItem[]>;
  deleteByPaymentId(payment_id: string): Promise<boolean>;
  deleteById(id: string): Promise<boolean>;
  findById(id: string): Promise<OrderItem | null>;
  update(id: string, updates: Partial<OrderItem>): Promise<OrderItem | null>;
}

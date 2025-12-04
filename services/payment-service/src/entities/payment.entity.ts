import {
  Column,
  DeleteDateColumn,
  Entity,
  OneToMany,
  PrimaryColumn,
} from 'typeorm';
import { OrderItem } from './order-item.entity';

@Entity('payments')
export class Payment {
  @PrimaryColumn('varchar')
  id: string;

  @Column('decimal', { precision: 12, scale: 2 })
  amount: number;

  @Column({ type: 'varchar', length: 255 })
  description: string;

  @Column({
    type: 'enum',
    enum: [
      'pending',
      'completed',
      'failed',
      'refunded',
      'cancelled',
      'expired',
    ],
    default: 'pending',
  })
  status: string;

  @Column({ type: 'varchar' })
  user_id: string;

  @Column({ type: 'varchar', length: 255, nullable: true })
  checkout_url: string | null;

  @Column({ type: 'varchar', length: 255, nullable: true })
  order_code: string | null;

  @Column({ type: 'varchar', length: 100, nullable: true })
  type: string | null;

  @Column({
    type: 'timestamp',
    default: () => "CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'",
  })
  created_at: Date;

  @Column({
    type: 'timestamp',
    default: () => "CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'",
    onUpdate: "CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'",
  })
  updated_at: Date;

  @DeleteDateColumn()
  deleted_at: Date | null;

  @Column({ type: 'timestamp', nullable: true })
  paid_at: Date | null;

  @Column({ type: 'timestamp', nullable: true })
  expired_at: Date | null;

  @OneToMany(() => OrderItem, (orderItem) => orderItem.payment_id, {
    cascade: true,
  })
  orderItems: OrderItem[];
}

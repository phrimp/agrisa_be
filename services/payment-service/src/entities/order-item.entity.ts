import {
  Entity,
  Column,
  PrimaryColumn,
  DeleteDateColumn,
  ManyToOne,
  JoinColumn,
} from 'typeorm';
import { Payment } from './payment.entity';

@Entity('order_items')
export class OrderItem {
  @PrimaryColumn('varchar')
  id: string;

  @ManyToOne(() => Payment, (payment: Payment) => payment.id, {
    onDelete: 'CASCADE',
  })
  @JoinColumn({ name: 'payment_id' })
  payment_id: string;

  @Column('varchar', { nullable: true })
  item_id: string | null;

  @Column('varchar')
  name: string;

  @Column('decimal', { precision: 12, scale: 2 })
  price: number;

  @Column('int', { default: 1 })
  quantity: number;

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
}

import {
  Entity,
  Column,
  PrimaryColumn,
  CreateDateColumn,
  UpdateDateColumn,
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

  @CreateDateColumn()
  created_at: Date;

  @UpdateDateColumn()
  updated_at: Date;

  @DeleteDateColumn()
  deleted_at: Date | null;
}

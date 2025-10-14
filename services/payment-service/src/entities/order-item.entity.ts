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

  @Column('varchar')
  item_id: string;

  @Column({ type: 'varchar', length: 255 })
  item_name: string;

  @Column('decimal', { precision: 12, scale: 2 })
  item_price: number;

  @Column({ type: 'varchar', length: 50, nullable: true })
  type: string | null;

  @CreateDateColumn()
  created_at: Date;

  @UpdateDateColumn()
  updated_at: Date;

  @DeleteDateColumn()
  deleted_at: Date | null;
}

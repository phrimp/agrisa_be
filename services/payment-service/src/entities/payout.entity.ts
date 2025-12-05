import {
  Column,
  DeleteDateColumn,
  Entity,
  JoinColumn,
  ManyToOne,
  PrimaryColumn,
} from 'typeorm';
import { Payment } from './payment.entity';

@Entity('payouts')
export class Payout {
  @PrimaryColumn('varchar')
  id: string;

  @Column('decimal', { precision: 12, scale: 2 })
  amount: number;

  @Column({ type: 'varchar', length: 255 })
  description: string;

  @Column({
    type: 'enum',
    enum: ['pending', 'completed'],
    default: 'pending',
  })
  status: string;

  @Column({ type: 'varchar' })
  user_id: string;

  @Column({ type: 'varchar', length: 255, nullable: true })
  bank_code: string | null;

  @Column({ type: 'varchar', length: 255, nullable: true })
  account_number: string | null;

  @Column({ type: 'varchar', length: 255, nullable: true })
  payment_id: string | null;

  @ManyToOne(() => Payment, (payment) => payment.id)
  @JoinColumn({ name: 'payment_id' })
  payment: Payment;

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
  completed_at: Date | null;
}

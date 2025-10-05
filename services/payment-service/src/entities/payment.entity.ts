import {
  Entity,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  PrimaryColumn,
  DeleteDateColumn,
} from 'typeorm';

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

  @CreateDateColumn()
  created_at: Date;

  @UpdateDateColumn()
  updated_at: Date;

  @DeleteDateColumn()
  deleted_at: Date | null;

  @Column({ type: 'timestamp', nullable: true })
  paid_at: Date | null;

  @Column({ type: 'timestamp', nullable: true })
  expired_at: Date | null;
}

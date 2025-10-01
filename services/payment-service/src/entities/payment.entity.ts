import {
  Entity,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  PrimaryColumn,
  DeleteDateColumn,
} from 'typeorm';
import { z } from 'zod';

export const createPaymentSchema = z.object({
  id: z.string(),
  amount: z.number().positive(),
  description: z.string().min(1).max(255),
  user_id: z.string(),
  order_code: z.string().max(255).nullable().optional(),
  status: z.enum(['pending', 'completed', 'failed', 'refunded']).optional(),
});

export const updatePaymentSchema = createPaymentSchema.partial();

export type CreatePaymentDto = z.infer<typeof createPaymentSchema>;
export type UpdatePaymentDto = z.infer<typeof updatePaymentSchema>;

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
    enum: ['pending', 'completed', 'failed', 'refunded'],
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
  expired_at: Date | null;
}

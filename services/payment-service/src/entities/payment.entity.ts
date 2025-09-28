import {
  Entity,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  PrimaryColumn,
} from 'typeorm';
import { z } from 'zod';

export const createPaymentSchema = z.object({
  id: z.string(),
  amount: z.number().positive(),
  description: z.string().min(1).max(255),
  userId: z.string(),
  transactionId: z.string().max(255).nullable().optional(),
  status: z.enum(['pending', 'completed', 'failed', 'refunded']).optional(),
});

export const updatePaymentSchema = createPaymentSchema.partial();

export type CreatePaymentDto = z.infer<typeof createPaymentSchema>;
export type UpdatePaymentDto = z.infer<typeof updatePaymentSchema>;

// TypeORM entity
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
  userId: string;

  @Column({ type: 'varchar', length: 255, nullable: true })
  transactionId: string | null;

  @CreateDateColumn()
  createdAt: Date;

  @UpdateDateColumn()
  updatedAt: Date;
}

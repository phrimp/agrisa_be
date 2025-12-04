import {
  Column,
  CreateDateColumn,
  Entity,
  PrimaryGeneratedColumn,
} from 'typeorm';

@Entity('notifications')
export class Notification {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column({ type: 'varchar', length: 255 })
  user_id: string;

  @Column({ type: 'varchar', length: 500 })
  title: string;

  @Column({ type: 'text' })
  body: string;

  @Column({ type: 'jsonb', nullable: true })
  data: any;

  @Column({ type: 'varchar', length: 50 })
  type: string; // 'expo' | 'web'

  @Column({ type: 'varchar', length: 50, default: 'sent' })
  status: string; // 'sent' | 'failed' | 'read'

  @Column({ type: 'text', nullable: true })
  error_message: string;

  @CreateDateColumn({ name: 'created_at' })
  created_at: Date;
}

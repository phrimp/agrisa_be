import {
  Column,
  CreateDateColumn,
  Entity,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';

@Entity('subscribers')
export class Subscriber {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column({ type: 'varchar', length: 255 })
  expo_token: string;

  @Column({ type: 'varchar', length: 255 })
  platform: string;

  @Column({ type: 'varchar', length: 255, nullable: true })
  p256dh: string;

  @Column({ type: 'varchar', length: 500, nullable: true })
  auth: string;

  @Column({ type: 'varchar', length: 500, nullable: true })
  endpoint: string;

  @Column({ type: 'varchar', length: 255 })
  user_id: string;

  @CreateDateColumn({ name: 'created_at' })
  created_at: Date;

  @UpdateDateColumn({ name: 'updated_at' })
  updated_at: Date;
}

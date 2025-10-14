import {
  Entity,
  Column,
  PrimaryColumn,
  CreateDateColumn,
  UpdateDateColumn,
  DeleteDateColumn,
} from 'typeorm';

@Entity('configurations')
export class Configuration {
  @PrimaryColumn('varchar')
  id: string;

  @Column('varchar', { nullable: false })
  payos_client_id: string;

  @Column('varchar', { nullable: false })
  payos_api_key: string;

  @Column('varchar', { nullable: false })
  payos_checksum_key: string;

  @Column('varchar', { nullable: true })
  payos_expired_duration?: string;

  @Column('int', { nullable: true })
  payos_order_code_length?: number;

  @Column('varchar', { nullable: true })
  payment_cron_expression?: string;

  @CreateDateColumn({ type: 'timestamp' })
  created_at: Date;

  @UpdateDateColumn({ type: 'timestamp' })
  updated_at: Date;

  @DeleteDateColumn({ type: 'timestamp', nullable: true })
  deleted_at?: Date;
}

-- Create database
CREATE DATABASE noti_service;

-- Note: After creating the database, connect to it and run the following commands

-- Create subscribers table
CREATE TABLE subscribers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  expo_token VARCHAR(255),
  platform VARCHAR(255) NOT NULL,
  p256dh VARCHAR(255),
  auth VARCHAR(500),
  endpoint VARCHAR(500),
  user_id VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create notifications table
CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id VARCHAR(255) NOT NULL,
  title VARCHAR(500) NOT NULL,
  body TEXT NOT NULL,
  data JSONB,
  platform VARCHAR(50) NOT NULL,
  status VARCHAR(50) DEFAULT 'sent',
  error_message TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Optional: Create indexes for better performance
CREATE INDEX idx_subscribers_user_id ON subscribers(user_id);
CREATE INDEX idx_subscribers_platform ON subscribers(platform);
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_platform ON notifications(platform);

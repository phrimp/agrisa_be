-- Push Notification Service Database Schema
-- Create database if not exists
CREATE DATABASE IF NOT EXISTS push_noti_service;

-- Create subscribers table for both Expo and Web Push subscriptions
CREATE TABLE IF NOT EXISTS subscribers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  expo_token VARCHAR(255) NULL, -- For Expo mobile app notifications
  type VARCHAR(255) NOT NULL, -- 'expo' for mobile app, 'web' for browser
  p256dh VARCHAR(255) NULL, -- Web push public key
  auth VARCHAR(500) NULL, -- Web push auth key
  endpoint VARCHAR(500) NULL, -- Web push endpoint URL
  user_id VARCHAR(255) NOT NULL, -- User identifier
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_subscribers_user_id ON subscribers(user_id);
CREATE INDEX IF NOT EXISTS idx_subscribers_type ON subscribers(type);
CREATE INDEX IF NOT EXISTS idx_subscribers_expo_token ON subscribers(expo_token) WHERE expo_token IS NOT NULL;

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_subscribers_updated_at
    BEFORE UPDATE ON subscribers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Sample data for testing (optional)
-- INSERT INTO subscribers (expo_token, type, user_id) VALUES
-- ('ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]', 'expo', 'user123'),
-- ('ExponentPushToken[yyyyyyyyyyyyyyyyyyyyyy]', 'expo', 'user456');

-- Web push subscription sample (endpoint, p256dh, auth would be actual values from browser)
-- INSERT INTO subscribers (type, endpoint, p256dh, auth, user_id) VALUES
-- ('web', 'https://fcm.googleapis.com/fcm/send/...', 'BNc...', 'tBH...', 'user789');

-- View to see active subscriptions
CREATE OR REPLACE VIEW active_subscriptions AS
SELECT
  id,
  user_id,
  type,
  CASE
    WHEN type = 'expo' THEN 'Mobile App (Expo)'
    WHEN type = 'web' THEN 'Web Browser'
    ELSE 'Unknown'
  END as device_type,
  created_at,
  updated_at
FROM subscribers
ORDER BY user_id, type, updated_at DESC;

-- Function to clean up expired subscriptions (optional)
CREATE OR REPLACE FUNCTION cleanup_expired_subscriptions()
RETURNS INTEGER AS $$
DECLARE
  deleted_count INTEGER;
BEGIN
  -- This is a placeholder - in a real implementation, you'd need to check
  -- with push services to see if subscriptions are still valid
  -- For now, just return 0
  deleted_count := 0;
  RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON DATABASE push_noti TO your_user;
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO your_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO your_user;
-- Bảng 1: insurance_partners
-- Lưu trữ thông tin về các đối tác bảo hiểm
CREATE TABLE insurance_partners (
    partner_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_name VARCHAR(255) NOT NULL,
    partner_logo_url TEXT,
    cover_photo_url TEXT,
    partner_tagline VARCHAR(500),
    partner_description TEXT,
    partner_phone VARCHAR(20),
    partner_email VARCHAR(100),
    partner_address TEXT,
    partner_website VARCHAR(255),
    partner_rating_score DECIMAL(2,1) CHECK (partner_rating_score >= 0 AND partner_rating_score <= 5),
    partner_rating_count INTEGER DEFAULT 0,
    trust_metric_experience INTEGER,
    trust_metric_clients INTEGER,
    trust_metric_claim_rate INTEGER CHECK (trust_metric_claim_rate >= 0 AND trust_metric_claim_rate <= 100),
    total_payouts TEXT,
    average_payout_time VARCHAR(255),
    confirmation_timeline VARCHAR(255),
    hotline VARCHAR(50),
    support_hours VARCHAR(255),
    coverage_areas TEXT,
    is_suspended BOOLEAN DEFAULT FALSE,
    suspended_at TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Bảng 2: products
-- Lưu trữ thông tin các gói bảo hiểm của từng đối tác
CREATE TABLE products (
    product_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id UUID NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    product_icon VARCHAR(100),
    product_description TEXT,
    product_supported_crop VARCHAR(50) CHECK (product_supported_crop IN ('lúa nước', 'cà phê')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (partner_id) REFERENCES insurance_partners(partner_id) ON DELETE CASCADE
);

-- Bảng 3: partner_reviews
-- Lưu trữ các đánh giá của nông dân về đối tác bảo hiểm
CREATE TABLE partner_reviews (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id UUID NOT NULL,
    reviewer_id UUID NOT NULL,
    reviewer_name VARCHAR(255) NOT NULL,
    reviewer_avatar_url TEXT,
    rating_stars INTEGER CHECK (rating_stars >= 1 AND rating_stars <= 5),
    review_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (partner_id) REFERENCES insurance_partners(partner_id) ON DELETE CASCADE
);

-- Tạo indexes để tối ưu performance
CREATE INDEX idx_products_partner_id ON products(partner_id);
CREATE INDEX idx_products_supported_crop ON products(product_supported_crop);
CREATE INDEX idx_reviews_partner_id ON partner_reviews(partner_id);
CREATE INDEX idx_reviews_rating_stars ON partner_reviews(rating_stars);
CREATE INDEX idx_partners_rating_score ON insurance_partners(partner_rating_score);

-- Trigger để tự động cập nhật updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_insurance_partners_updated_at BEFORE UPDATE ON insurance_partners
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_partner_reviews_updated_at BEFORE UPDATE ON partner_reviews
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- User profile
CREATE TABLE user_profiles (
  -- Identity
  profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id VARCHAR(255) NOT NULL UNIQUE, -- From Auth Service
  role_id VARCHAR(255) NOT NULL, -- From Auth Service (farmer/staff/admin)
  
  -- Company Association (NULL for farmers, populated for insurance staff)
  partner_id UUID, -- FK to insurance_partners
  
  -- Basic Personal Information
  full_name VARCHAR(255) NOT NULL,
  display_name VARCHAR(100),
  date_of_birth DATE,
  gender VARCHAR(20),
  nationality VARCHAR(10) DEFAULT 'VN',
  
  -- Contact Information
  primary_phone VARCHAR(20) NOT NULL,
  alternate_phone VARCHAR(20),
  email VARCHAR(255),
  
  -- Address Information (Vietnamese Format)
  permanent_address TEXT,
  current_address TEXT,
  province_code VARCHAR(10), -- e.g., "79" for HCMC
  province_name VARCHAR(100),
  district_code VARCHAR(10),
  district_name VARCHAR(100),
  ward_code VARCHAR(10),
  ward_name VARCHAR(100),
  postal_code VARCHAR(10),
  
  -- Metadata
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  last_updated_by VARCHAR(255), -- user_id from Auth Service
  
  CONSTRAINT unique_user_id UNIQUE(user_id),
  CONSTRAINT fk_company FOREIGN KEY (partner_id) 
    REFERENCES insurance_partners(partner_id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX idx_user_profile_user_id ON user_profiles(user_id);
CREATE INDEX idx_user_profile_role_id ON user_profiles(role_id);
CREATE INDEX idx_user_profile_company ON user_profiles(partner_id);
CREATE INDEX idx_user_profile_phone ON user_profiles(primary_phone);
CREATE INDEX idx_user_profile_email ON user_profiles(email);
CREATE INDEX idx_user_profile_province ON user_profiles(province_code);

-- Ví dụ INSERT data mẫu
INSERT INTO insurance_partners (
    partner_id, partner_name, partner_logo_url, cover_photo_url,
    partner_tagline, partner_description, partner_phone, partner_email,
    partner_address, partner_website, partner_rating_score, partner_rating_count,
    trust_metric_experience, trust_metric_clients, trust_metric_claim_rate,
    total_payouts, average_payout_time, confirmation_timeline, hotline,
    support_hours, coverage_areas
) VALUES (
    'PARTNER001',
    'Bảo Hiểm An Tâm',
    'https://example.com/logo.png',
    'https://example.com/cover.jpg',
    'Đồng hành cùng nhà nông, vững tâm sản xuất',
    'Nhà cung cấp bảo hiểm nông nghiệp hàng đầu tại Việt Nam, chuyên về bảo hiểm tham số cho cây trồng được hỗ trợ bởi công nghệ vệ tinh.',
    '+84 123 456 789',
    'info@antaminsurance.com',
    '123 Nguyen Hue Street, District 1, Ho Chi Minh City, Vietnam',
    'https://www.antaminsurance.com',
    4.8,
    1256,
    15,
    20000,
    98,
    'Khoảng 3 tỷ VND',
    '3 ngày làm việc sau khi xác nhận thanh toán',
    'Trong vòng 24 giờ (chậm nhất là 48 giờ)',
    '+84 1800 123 456 (24/7)',
    'Thứ 2 đến thứ 6, 8 AM - 6 PM, cuối tuần chỉ nhận hỗ trợ qua email',
    'An Giang, Cà Mau, Đồng Tháp'
);

INSERT INTO products (
    product_id, partner_id, product_name, product_icon,
    product_description, product_supported_crop
) VALUES 
(
    'PROD001',
    'PARTNER001',
    'Bảo hiểm Cây Lúa',
    'fa-solid fa-wheat-awn',
    'Bảo vệ toàn diện trước rủi ro thiên tai, sâu bệnh.',
    'lúa nước'
),
(
    'PROD002',
    'PARTNER001',
    'Bảo hiểm Cây Cà Phê',
    'fa-solid fa-coffee-beans',
    'Bảo vệ mùa màng cà phê trước biến đổi khí hậu và dịch bệnh.',
    'cà phê'
);

INSERT INTO partner_reviews (
    review_id, partner_id, reviewer_id, reviewer_name,
    reviewer_avatar_url, rating_stars, review_content
) VALUES (
    'REV001',
    'PARTNER001',
    'UCtWFgbIha',
    'Anh Bảy',
    'https://example.com/avatar1.jpg',
    5,
    'Nhờ có khoản bồi thường kịp thời từ Bảo hiểm An Tâm mà gia đình tôi đã có vốn để tái sản xuất sau đợt ngập lụt vừa rồi. Thủ tục rất nhanh gọn, nhân viên nhiệt tình.'
);

INSERT INTO partner_reviews (
    partner_id, reviewer_id, reviewer_name,
    reviewer_avatar_url, rating_stars, review_content,
    created_at, updated_at
) VALUES
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '2ccb00c6-7762-455a-8917-f687690303e0', 'Anh Bảy', 'https://example.com/avatar1.jpg', 5, 'Nhờ có khoản bồi thường kịp thời từ Bảo hiểm An Tâm mà gia đình tôi đã có vốn để tái sản xuất sau đợt ngập lụt vừa rồi. Thủ tục rất nhanh gọn, nhân viên nhiệt tình.', '2025-10-10 11:24:22.250', '2025-10-10 11:24:22.250'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '0d23ca38-f1b8-4e9b-a92b-5f148f4aa365', 'Anh Bảy', 'https://example.com/avatar1.jpg', 5, 'Nhờ có khoản bồi thường kịp thời từ Bảo hiểm An Tâm mà gia đình tôi đã có vốn để tái sản xuất sau đợt ngập lụt vừa rồi. Thủ tục rất nhanh gọn, nhân viên nhiệt tình.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', 'ec518676-e351-487a-a449-d954ca4849e3', 'Chị Hoa', 'https://example.com/avatar2.jpg', 4, 'Dịch vụ tốt, nhân viên tư vấn rõ ràng, tuy nhiên thời gian xử lý hồ sơ còn hơi lâu một chút.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '7dbb39e3-36f6-47c3-9167-65eafcc17c95', 'Anh Minh', 'https://example.com/avatar3.jpg', 5, 'Bảo hiểm An Tâm rất đáng tin cậy. Tôi đã được chi trả đúng cam kết sau khi ruộng bị thiệt hại do sâu bệnh.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '42208f68-76f2-4dd1-bf2a-3373928b126a', 'Cô Hạnh', 'https://example.com/avatar4.jpg', 3, 'Tôi mong bên bảo hiểm cải thiện khâu liên hệ khách hàng, đôi khi khó gọi điện để hỏi thông tin.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '8d15475e-1a10-4b97-b516-a943be234c25', 'Anh Tâm', 'https://example.com/avatar5.jpg', 4, 'Dịch vụ khá ổn, ứng dụng dễ dùng, chỉ cần cải thiện tốc độ phản hồi của tổng đài.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', 'ad9e98b0-ef91-4205-8145-578624d5e2fe', 'Chị Lan', 'https://example.com/avatar6.jpg', 5, 'Rất hài lòng! Hồ sơ xử lý nhanh, nhân viên hỗ trợ tận tâm, sẽ tiếp tục tham gia năm sau.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '2a2f0952-8a25-47a5-8a3d-5056147477b4', 'Anh Dũng', 'https://example.com/avatar7.jpg', 2, 'Tôi phải chờ khá lâu mới được phản hồi. Hy vọng bên công ty cải thiện quy trình chăm sóc khách hàng.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', 'b9fef998-adc0-4580-a05d-09cd986cba88', 'Chú Năm', 'https://example.com/avatar8.jpg', 5, 'Nhân viên thân thiện, giải thích kỹ càng, giúp tôi hiểu rõ quyền lợi khi tham gia bảo hiểm cây trồng.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '8fe95d39-e0f9-43e7-998a-d4c748ea7ab1', 'Anh Phong', 'https://example.com/avatar9.jpg', 4, 'Mức phí hợp lý, thông tin minh bạch. Tôi đã giới thiệu cho nhiều người trong xã cùng tham gia.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '687c78e3-7d54-43d0-a18e-ecffb161ed70', 'Cô Tư', 'https://example.com/avatar10.jpg', 5, 'Tôi đánh giá cao sự chuyên nghiệp của đội ngũ hỗ trợ. Hồ sơ được giải quyết nhanh chóng, chính xác.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '1aaa7bcd-c36a-442f-b408-a4030fb9f00b', 'Anh Quang', 'https://example.com/avatar11.jpg', 3, 'Dịch vụ ổn nhưng cần thêm kênh hỗ trợ trực tuyến để người dân dễ dàng tra cứu thông tin.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '10fb023a-7c2d-4d4c-b03f-e7943f94533e', 'Chị Mai', 'https://example.com/avatar12.jpg', 5, 'Thủ tục tham gia đơn giản, phí hợp lý, hỗ trợ rất chu đáo. Tôi rất yên tâm khi đồng hành cùng công ty.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508'),
('623be0ba-1775-473b-8cae-a69f1a2ecd61', '53281e30-ab6e-47c3-bda4-146198fd6137', 'Bác Sáu', 'https://example.com/avatar13.jpg', 4, 'Sau đợt hạn hán vừa rồi, công ty đã bồi thường đúng hạn. Tôi rất cảm kích sự hỗ trợ kịp thời.', '2025-10-15 09:43:37.508', '2025-10-15 09:43:37.508');

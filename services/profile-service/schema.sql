-- Bảng 1: insurance_partners
-- Lưu trữ thông tin về các đối tác bảo hiểm
CREATE TABLE insurance_partners (
    partner_id VARCHAR(50) PRIMARY KEY,
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
    product_id VARCHAR(50) PRIMARY KEY,
    partner_id VARCHAR(50) NOT NULL,
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
    review_id VARCHAR(50) PRIMARY KEY,
    partner_id VARCHAR(50) NOT NULL,
    reviewer_id VARCHAR(50) NOT NULL,
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
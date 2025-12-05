-- enum
CREATE TYPE deletion_request_status AS ENUM ('pending', 'approved', 'rejected', 'cancelled', 'completed');

CREATE TABLE insurance_partners (
    partner_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    legal_company_name VARCHAR(255) NOT NULL,
    partner_trading_name VARCHAR(255),
    partner_display_name VARCHAR(255),
    partner_logo_url TEXT,
    cover_photo_url TEXT,
    company_type VARCHAR(50),
    incorporation_date DATE,
    tax_identification_number VARCHAR(50) UNIQUE NOT NULL,
    business_registration_number VARCHAR(100) UNIQUE NOT NULL,
    partner_tagline VARCHAR(500),
    partner_description TEXT,
    partner_phone VARCHAR(20),
    partner_official_email VARCHAR(100),
    head_office_address TEXT NOT NULL,
    province_code VARCHAR,
    province_name VARCHAR,
    ward_code VARCHAR,
    ward_name VARCHAR,
    postal_code VARCHAR(10),
    fax_number VARCHAR(20),
    customer_service_hotline VARCHAR(20),
    insurance_license_number VARCHAR(100) UNIQUE,
    license_issue_date DATE,
    license_expiry_date DATE,
    authorized_insurance_lines TEXT[],
    operating_provinces TEXT[],
    year_established INTEGER,
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
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'suspended', 'terminated', 'under_review')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_by_id VARCHAR,
    last_updated_by_name VARCHAR,
    legal_document_urls TEXT[] DEFAULT ARRAY[]::TEXT[]
);

-- -- Bảng 2: products
-- -- Lưu trữ thông tin các gói bảo hiểm của từng đối tác
-- CREATE TABLE products (
--     product_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     partner_id UUID NOT NULL,
--     product_name VARCHAR(255) NOT NULL,
--     product_icon VARCHAR(100),
--     product_description TEXT,
--     product_supported_crop VARCHAR(50) CHECK (product_supported_crop IN ('lúa nước', 'cà phê')),
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     FOREIGN KEY (partner_id) REFERENCES insurance_partners(partner_id) ON DELETE CASCADE
-- );

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
CREATE INDEX idx_reviews_partner_id ON partner_reviews(partner_id);
CREATE INDEX idx_reviews_rating_stars ON partner_reviews(rating_stars);
CREATE INDEX idx_partners_rating_score ON insurance_partners(partner_rating_score);

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

  -- Bank info
  account_number VARCHAR,
  account_name VARCHAR,
  bank_code VARCHAR;
  
  -- Metadata
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  last_updated_by VARCHAR(255), -- user_id from Auth Service
  last_updated_by_name VARCHAR(255),
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

-- Create partner_deletion_requests table
CREATE TABLE partner_deletion_requests (
    -- Primary key
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Partner reference
    partner_id UUID NOT NULL,
    
    -- Requester information
    requested_by VARCHAR(255) NOT NULL, -- User ID of partner admin
    requested_by_name VARCHAR(255) NOT NULL,
    
    -- Request details
    detailed_explanation TEXT,
    
    -- Status and timeline
    status deletion_request_status NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    cancellable_until TIMESTAMP NOT NULL,
    
    -- Reviewer information (người duyệt/từ chối)
    reviewed_by_id VARCHAR(255),          -- User ID của người duyệt
    reviewed_by_name VARCHAR(255),        -- Tên người duyệt
    reviewed_at TIMESTAMP,                -- Thời gian duyệt/từ chối
    review_note TEXT,                     -- Ghi chú của người duyệt (optional)
    
    -- Metadata
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Foreign key constraint
    CONSTRAINT fk_partner 
        FOREIGN KEY (partner_id) 
        REFERENCES insurance_partners(partner_id) 
        ON DELETE CASCADE,
    
    -- Ensure cancellable_until is after requested_at
    CONSTRAINT check_cancellable_period 
        CHECK (cancellable_until > requested_at)
);

-- Create indexes for better query performance
CREATE INDEX idx_deletion_requests_partner_id ON partner_deletion_requests(partner_id);
CREATE INDEX idx_deletion_requests_status ON partner_deletion_requests(status);
CREATE INDEX idx_deletion_requests_requested_by ON partner_deletion_requests(requested_by);
CREATE INDEX idx_deletion_requests_requested_at ON partner_deletion_requests(requested_at);
CREATE INDEX idx_deletion_requests_cancellable_until ON partner_deletion_requests(cancellable_until);

-- Optional: Add comment to table
COMMENT ON TABLE partner_deletion_requests IS 'Stores partner deletion requests with cancellation period';
COMMENT ON COLUMN partner_deletion_requests.cancellable_until IS 'Deadline for cancelling the deletion request (requested_at + x days)';

-- Ví dụ INSERT data mẫu
INSERT INTO insurance_partners (
    legal_company_name,
    partner_trading_name,
    partner_display_name,
    partner_logo_url,
    cover_photo_url,
    company_type,
    incorporation_date,
    tax_identification_number,
    business_registration_number,
    partner_tagline,
    partner_description,
    partner_phone,
    partner_official_email,
    head_office_address,
    province_code,
    province_name,
    ward_code,
    ward_name,
    postal_code,
    fax_number,
    customer_service_hotline,
    insurance_license_number,
    license_issue_date,
    license_expiry_date,
    authorized_insurance_lines,
    operating_provinces,
    year_established,
    partner_website,
    partner_rating_score,
    partner_rating_count,
    trust_metric_experience,
    trust_metric_clients,
    trust_metric_claim_rate,
    total_payouts,
    average_payout_time,
    confirmation_timeline,
    hotline,
    support_hours,
    coverage_areas,
    status
) VALUES 
(
    'Công ty Cổ phần Bảo hiểm An Tâm Nông Nghiệp',
    'Bảo hiểm An Tâm',
    'An Tâm Insurance',
    'https://cdn.agrisa.vn/logos/antam-insurance-logo.png',
    'https://cdn.agrisa.vn/covers/antam-cover-rice-field.jpg',
    'domestic',
    '2008-03-15',
    '0123456789',
    '0108123456',
    'Đồng hành cùng nhà nông, vững tâm sản xuất',
    'Nhà cung cấp bảo hiểm nông nghiệp hàng đầu tại Việt Nam, chuyên về bảo hiểm tham số cho cây trồng được hỗ trợ bởi công nghệ vệ tinh. Với hơn 15 năm kinh nghiệm, chúng tôi cam kết bảo vệ người nông dân trước các rủi ro thiên tai và sâu bệnh.',
    '+84 28 3822 1234',
    'info@antaminsurance.com.vn',
    '145 Pasteur, Phường Bến Nghé, Quận 1, Thành phố Hồ Chí Minh',
    '79',
    'Thành phố Hồ Chí Minh',
    '26734',
    'Phường Bến Nghé',
    '700000',
    '+84 28 3822 1235',
    '1800 1234 (24/7)',
    'BH-NN-2008-001234',
    '2008-04-01',
    '2028-04-01',
    ARRAY['agricultural', 'crop_insurance', 'parametric_insurance'],
    ARRAY['An Giang', 'Đồng Tháp', 'Cần Thơ', 'Sóc Trăng', 'Bạc Liêu', 'Cà Mau', 'Kiên Giang', 'Long An', 'Tiền Giang', 'Vĩnh Long', 'Trà Vinh', 'Bến Tre', 'Hậu Giang'],
    2008,
    'https://www.antaminsurance.com.vn',
    4.8,
    1256,
    15,
    20000,
    98,
    'Khoảng 3 tỷ VND',
    '3 ngày làm việc sau khi xác nhận thanh toán',
    'Trong vòng 24 giờ (chậm nhất là 48 giờ)',
    '+84 1800 1234 (24/7)',
    'Thứ 2 đến Chủ nhật, 7:00 - 22:00',
    'An Giang, Cà Mau, Đồng Tháp, Cần Thơ, Sóc Trăng, Kiên Giang, Long An',
    'active'
),
(
    'Công ty TNHH Bảo hiểm Nông Nghiệp Việt',
    'Bảo hiểm Nông Nghiệp Việt',
    'Việt Agri Insurance',
    'https://cdn.agrisa.vn/logos/viet-agri-logo.png',
    'https://cdn.agrisa.vn/covers/viet-agri-coffee-plantation.jpg',
    'domestic',
    '2012-07-20',
    '0234567890',
    '0109234567',
    'Bảo vệ mùa màng, yên tâm tương lai',
    'Công ty bảo hiểm chuyên về bảo hiểm cây trồng và nông nghiệp công nghệ cao. Chúng tôi cung cấp giải pháp bảo hiểm linh hoạt với quy trình thanh toán nhanh chóng, được hàng ngàn nông dân tin tưởng trên toàn quốc.',
    '+84 24 3944 5678',
    'contact@vietagriinsurance.vn',
    '28 Trần Hưng Đạo, Phường Phan Chu Trinh, Quận Hoàn Kiếm, Hà Nội',
    '01',
    'Hà Nội',
    '00265',
    'Phường Phan Chu Trinh',
    '100000',
    '+84 24 3944 5679',
    '1900 5678 (24/7)',
    'BH-NN-2012-005678',
    '2012-08-15',
    '2027-08-15',
    ARRAY['agricultural', 'crop_insurance', 'livestock_insurance'],
    ARRAY['Hà Nội', 'Hải Phòng', 'Quảng Ninh', 'Hải Dương', 'Hưng Yên', 'Thái Bình', 'Nam Định', 'Ninh Bình', 'Thanh Hóa', 'Nghệ An', 'Hà Tĩnh', 'Đắk Lắk', 'Lâm Đồng'],
    2012,
    'https://www.vietagriinsurance.vn',
    4.6,
    842,
    11,
    15000,
    95,
    'Khoảng 2.5 tỷ VND',
    '5 ngày làm việc sau khi xác nhận thanh toán',
    'Trong vòng 48 giờ',
    '+84 1900 5678 (24/7)',
    'Thứ 2 đến Thứ 6, 8:00 - 18:00, Thứ 7 8:00 - 12:00',
    'Thanh Hóa, Nghệ An, Hà Tĩnh, Đắk Lắk, Lâm Đồng',
    'active'
),
(
    'Công ty Cổ phần Bảo hiểm Đồng Bằng',
    'Bảo hiểm Đồng Bằng',
    'Đồng Bằng Insurance',
    'https://cdn.agrisa.vn/logos/dongbang-insurance-logo.png',
    'https://cdn.agrisa.vn/covers/dongbang-delta-landscape.jpg',
    'domestic',
    '2015-11-10',
    '0345678901',
    '0110345678',
    'Chắp cánh ước mơ cánh đồng xanh',
    'Đối tác tin cậy của người nông dân đồng bằng sông Cửu Long. Chuyên cung cấp bảo hiểm tham số cho lúa và cây trồng ngắn ngày với mức phí cạnh tranh, thanh toán bồi thường nhanh chóng dựa trên dữ liệu thời tiết chính xác.',
    '+84 292 3555 888',
    'info@dongbanginsurance.com.vn',
    '56 Nguyễn Văn Linh, Phường An Khánh, Quận Ninh Kiều, Thành phố Cần Thơ',
    '92',
    'Thành phố Cần Thơ',
    '31117',
    'Phường An Khánh',
    '900000',
    '+84 292 3555 889',
    '1800 6789 (24/7)',
    'BH-NN-2015-006789',
    '2015-12-01',
    '2030-12-01',
    ARRAY['agricultural', 'crop_insurance', 'parametric_insurance', 'weather_index_insurance'],
    ARRAY['Cần Thơ', 'An Giang', 'Đồng Tháp', 'Tiền Giang', 'Vĩnh Long', 'Bến Tre', 'Trà Vinh', 'Sóc Trăng', 'Hậu Giang', 'Bạc Liêu', 'Cà Mau', 'Kiên Giang', 'Long An'],
    2015,
    'https://www.dongbanginsurance.com.vn',
    4.7,
    1089,
    8,
    18500,
    97,
    'Khoảng 2.8 tỷ VND',
    '2 ngày làm việc sau khi xác nhận thanh toán',
    'Trong vòng 24 giờ',
    '+84 1800 6789 (24/7)',
    'Thứ 2 đến Chủ nhật, 6:00 - 21:00',
    'An Giang, Đồng Tháp, Cần Thơ, Tiền Giang, Vĩnh Long, Bến Tre, Sóc Trăng, Hậu Giang',
    'active'
);

INSERT INTO products (
    partner_id, product_name, product_icon,
    product_description, product_supported_crop
) VALUES 
(
    'c2f2aaaf-c3d5-4ea0-95f8-c6ce5d82a059',
    'Bảo hiểm Cây Lúa',
    'fa-solid fa-wheat-awn',
    'Bảo vệ toàn diện trước rủi ro thiên tai, sâu bệnh.',
    'lúa nước'
),
(
    'c2f2aaaf-c3d5-4ea0-95f8-c6ce5d82a059',
    'Bảo hiểm Cây Cà Phê',
    'fa-solid fa-coffee-beans',
    'Bảo vệ mùa màng cà phê trước biến đổi khí hậu và dịch bệnh.',
    'cà phê'
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

--User Profiles sample data
INSERT INTO user_profiles (
    user_id,
    role_id,
    partner_id,
    full_name,
    display_name,
    date_of_birth,
    gender,
    nationality,
    email,
    primary_phone,
    alternate_phone,
    permanent_address,
    current_address,
    province_code,
    province_name,
    district_code,
    district_name,
    ward_code,
    ward_name,
    postal_code,
    last_updated_by,
    last_updated_by_name
) VALUES 
-- Record 01: Farmer - Hoang Duy Giap
(
    '6956c836-fbe1-44d7-8236-d29f55cfb75e',
    'farmer',
    NULL,
    'Hoang Duy Giap',
    'Zapz31',
    '2004-01-31',
    'M',
    'VN',
    'hgiap46@gmail.com',
    '+84867931488',
    '+84909123456',
    '123 Đường Lê Văn Việt, Phường Tăng Nhơn Phú A, Thành phố Thủ Đức, Thành phố Hồ Chí Minh',
    '123 Đường Lê Văn Việt, Phường Tăng Nhơn Phú A, Thành phố Thủ Đức, Thành phố Hồ Chí Minh',
    '79',
    'Thành phố Hồ Chí Minh',
    '783',
    'Thành phố Thủ Đức',
    '27145',
    'Phường Tăng Nhơn Phú A',
    '700000',
    'aae1a169-ea85-4edf-bfcb-c7c1961cb357',
    'Admin'
),
-- Record 02: Staff - Vo Thanh Nhan
(
    '7ac64de7-47dc-46e9-8980-bf1113b91efd',
    'staff',
    '34b654f7-9509-4978-b9e8-28c4244d63e8',
    'Vo Thanh Nhan',
    'Tnhaan20',
    '2004-07-20',
    'M',
    'VN',
    'nhan@gmail.com',
    '+84867933333',
    '+84908234567',
    '456 Đường Nguyễn Văn Linh, Phường Tân Phú, Quận 7, Thành phố Hồ Chí Minh',
    '456 Đường Nguyễn Văn Linh, Phường Tân Phú, Quận 7, Thành phố Hồ Chí Minh',
    '79',
    'Thành phố Hồ Chí Minh',
    '778',
    'Quận 7',
    '27259',
    'Phường Tân Phú',
    '700000',
    'aae1a169-ea85-4edf-bfcb-c7c1961cb357',
    'Admin'
),
-- Record 03: Farmer - Lai Chi Thinh
(
    '5de65c29-ba32-4f49-ad8b-95bf4a0448a2',
    'farmer',
    NULL,
    'Lai Chi Thinh',
    'Prosper de Laval',
    '2004-10-13',
    'M',
    'VN',
    'thinh@gmail.com',
    '+84867944444',
    '+84907345678',
    '789 Đường Võ Văn Ngân, Phường Linh Chiểu, Thành phố Thủ Đức, Thành phố Hồ Chí Minh',
    '789 Đường Võ Văn Ngân, Phường Linh Chiểu, Thành phố Thủ Đức, Thành phố Hồ Chí Minh',
    '79',
    'Thành phố Hồ Chí Minh',
    '783',
    'Thành phố Thủ Đức',
    '27169',
    'Phường Linh Chiểu',
    '700000',
    'aae1a169-ea85-4edf-bfcb-c7c1961cb357',
    'Admin'
);

INSERT INTO partner_deletion_requests (
    partner_id,
    requested_by,
    requested_by_name,
    detailed_explanation,
    status,
    requested_at,
    cancellable_until
) VALUES (
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::UUID, 
    'user_123456',                                   
    'Nguyễn Văn A',                                 
    'Công ty chúng tôi quyết định ngừng hoạt động kinh doanh bảo hiểm nông nghiệp và muốn xóa tài khoản đối tác.',
    'pending',                                       
    NOW(),                                         
    NOW() + INTERVAL '7 days'                       
);
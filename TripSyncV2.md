# ✈️ Project Context: TripSync (Web Version)

## 1. Technical Stack & Architecture
* **Frontend**: React, TypeScript (Tuyệt đối **không** sử dụng Tailwind CSS).
* **Backend (Go)**: Kiến trúc phân lớp (Clean Architecture style):
    * **Route**: Khai báo các API endpoints.
    * **Controller**: Xử lý Request/Response và Validation.
    * **UseCase**: Chứa Business Logic cốt lõi.
    * **Repository**: Thao tác trực tiếp với Database (PostgreSQL).
* **Platform**: Web Application (Hỗ trợ PWA cho truy cập Offline).
* **Real-time**: Đồng bộ hóa tức thời qua WebSocket hoặc Firebase.
* **External APIs / 3rd Party Integration**: 
    * **Gemini API**: AI tạo đề xuất lịch trình tự động (Tour Guide).
    * **Agoda (Crawl/API)**: Trích xuất dữ liệu phòng khách sạn và giá cả thực tế (Receptionist).

## 2. Core Modules (Business Logic)
### 2.1. Authentication & Group Management
* **Auth**: Đăng nhập (Email/Social), Đăng ký và Quên mật khẩu.
* **Group Dashboard**: 
    * Hiển thị danh sách nhóm.
    * **Tạo nhóm mới (Owner)**: Nhập thông tin chi tiết để cấu hình chuyến đi:
        * Tên nhóm & Mô tả.
        * Ngày bắt đầu & Ngày kết thúc.
        * **[MỚI] Hành trình (Route Destinations)**: Danh sách các điểm đến (VD: Hà Nội, Quảng Bình, Đà Nẵng).
        * **[MỚI] Gu lưu trú (Accommodation Preference)**: Lựa chọn Khách sạn (HOTEL), Cắm trại (CAMPING), hoặc Linh hoạt (MIXED).
        * **[MỚI] Số lượng thành viên dự kiến**.
        * **[MỚI] Ngân sách dự kiến cho 1 người**.
        * *Logic Tích hợp AI (Hybrid)*: Khi submit, hệ thống gọi **Gemini API** truyền vào toàn bộ tham số trên. Gemini sẽ lên lịch trình chi tiết và trả về JSON.
            * Nếu đề xuất là Đi chơi/Ăn uống/Camping: Lưu thẳng vào DB.
            * Nếu đề xuất là Ngủ Khách sạn (HOTEL): Lấy ngân sách và địa điểm ngày hôm đó gọi **Agoda API** tìm 3-5 khách sạn có thật.
            * Toàn bộ data (Gemini + Agoda) được insert vào bảng `ACTIVITIES` với trạng thái `PENDING`.
        * Tự động sinh ID nhóm để chia sẻ.
    * Tham gia nhóm (Member) bằng ID được chia sẻ.

### 2.2. Lịch trình cộng tác (Itinerary)
* **Giao diện**: Hiển thị Timeline/Grid theo ngày bằng MUI.
* **Cơ chế Đề xuất & Vote**: 
    * Các đề xuất AI/Agoda sinh ra sẽ nằm ở khu vực "Đang bỏ phiếu" (`PENDING`).
    * User có thể xem được ảnh thumbnail, giá tiền, rating (đối với khách sạn).
    * Mọi thành viên đều có thể vote (👍) hoặc đề xuất thêm hoạt động mới.
    * Hoạt động nhiều vote nhất sẽ được Owner chốt vào lịch chính thức (`APPROVED`).
* **Tính năng bổ trợ**: Tích hợp Google Maps và Drag & Drop để sắp xếp lịch trình.

### 2.3. Chia tiền thông minh (Smart Bill Splitter)
* **Chức năng**: Thêm chi tiêu, hỗ trợ đa tiền tệ.
* **Cơ chế chia tiền**: Chia đều, theo tỷ lệ, hoặc **tick chọn từng thành viên cụ thể** tham gia khoản chi.
* **Output**: Tự động tính toán bảng cân đối "ai nợ ai" và hướng dẫn thanh toán.

### 2.4. Document Vault & Checklist
* **Tài liệu**: Lưu trữ vé máy bay, xác nhận khách sạn; hỗ trợ xem offline.
* **Checklist**: Danh sách đồ dùng cần mang và phân công nhiệm vụ cho từng người.

## 3. Permissions & Rules
* **Owner (Người tạo)**: Toàn quyền quản lý thành viên, chỉnh sửa thông tin nhóm, và chốt các nội dung (Lịch trình/Chi tiêu).
* **Member (Thành viên)**: Đề xuất, bình chọn, thêm chi tiêu và thực hiện nhiệm vụ được giao.
* **Export**: Có tính năng xuất toàn bộ chuyến đi thành link chia sẻ nhanh.
* **Admin Screen**: Có màn hình quản trị riêng cho các thiết lập nâng cao, như quản lý toàn bộ các nhóm, bảng thống kê, quản lý user.

## 4. Coding Standard
* **Go**: Tuân thủ Error Handling (`if err != nil`) và tối ưu concurrency.
* **React**: Sử dụng Functional Components và Hooks; Type-safe tuyệt đối với TypeScript.
* **UI/UX**: Không dùng Tailwind; sử dụng **MUI (Material-UI)**, ưu tiên tính trực quan, thao tác nhanh và hover tooltips.
* **Grid Layout**: Sử dụng Grid MUI bản mới (không truyền prop `item`).
* **Database**: Sử dụng `*gorm.DB` để tương tác với PostgreSQL.

## 5. Current Progress
- Đã xong(đây là những phần việc trước khi thêm nghiệp vụ): Login, Register, tạo nhóm, join nhóm, phân quyền thành viên, CRUD Activity, Vote, Finalize.
- Đã xong(đây là những phần việc sau khi có nghiệp vụ mới): sửa lại models để migrate lại database.
- Đang làm: Cập nhật DB cho tính năng AI Planning, viết luồng gọi Hybrid API (Gemini + Agoda) xử lý đa điểm đến và gu lưu trú (MIXED).

## 6. Third-Party Data Structs (Agoda)
Cấu trúc dữ liệu thu thập từ Agoda phục vụ cho module khách sạn:

```go
// AgodaResponse là struct bọc ngoài cùng
type AgodaResponse struct {
	Data struct {
		CitySearch struct {
			Properties []AgodaHotel `json:"properties"`
		} `json:"citySearch"`
	} `json:"data"`
}

// AgodaHotel chứa thông tin khách sạn
type AgodaHotel struct {
	Content struct {
		InformationSummary struct {
			LocaleName string  `json:"localeName"` // 1. Tên khách sạn
			Rating     float64 `json:"rating"`     // 2. Số sao
		} `json:"informationSummary"`
		GeoInfo struct {
			Latitude  float64 `json:"latitude"`  
			Longitude float64 `json:"longitude"` 
		} `json:"geoInfo"`
		Images struct {
			HotelImages []struct {
				Urls []struct {
					Value string `json:"value"` // 4. Link ảnh thumbnail
				} `json:"urls"`
			} `json:"hotelImages"`
		} `json:"images"`
	} `json:"content"`

	Pricing struct {
		Offers []struct {
			RoomOffers []struct {
				Room struct {
					Pricing []struct {
						Currency string `json:"currency"` // 5a. Loại tiền
						Price    struct {
							PerRoomPerNight struct {
								Inclusive struct {
									Display float64 `json:"display"` // 5b. Giá tiền 1 đêm
								} `json:"inclusive"`
							} `json:"perRoomPerNight"`
						} `json:"price"`
					} `json:"pricing"`
				} `json:"room"`
			} `json:"roomOffers"`
		} `json:"offers"`
	} `json:"pricing"`
}
```

## 7 database:

---
config:
  layout: elk
  theme: neutral
---
erDiagram
	direction LR
	USERS {
		id int PK ""  
		full_name string  ""  
		email string  ""  
		password string  ""  
		avatar string  ""  
		role string  ""  
	}

	GROUP_MEMBERS {
		group_id int PK,FK ""  
		user_id int PK,FK ""  
		role string  "ADMIN / MEMBER"  
		joined_at datetime  ""  
	}

	GROUPS {
		id int PK ""  
		name string  ""  
		description string  ""  
		start_date datetime  ""  
		end_date datetime  ""  
		invite_code string  ""  
		is_public boolean  ""  
		share_token string  ""  
		route_destinations string  "Danh sách điểm đến (VD: Hà Nội, Huế)"  
		accommodation_pref string  "Gu lưu trú (HOTEL / CAMPING / MIXED)"  
		expected_members int  "Số người dự kiến"  
		budget_per_person decimal  "Ngân sách/người"  
		currency string  "Loại tiền tệ (VND)"  
	}

	ACTIVITIES {
		id int PK ""  
		group_id int FK ""  
		name string  ""  
		type string  "HOTEL / ATTRACTION / RESTAURANT / CAMPING"  
		location string  ""  
		description string  ""  
		start_time datetime  ""  
		end_time datetime  ""  
		status string  "PENDING / APPROVED"  
		created_by int FK "Null nếu là AI tạo"  
		lat float  ""  
		lng float  ""  
		place_id int  ""  
		is_ai_generated boolean  "Đánh dấu AI/Agoda tạo"  
		estimated_cost decimal  "Chi phí dự kiến"  
		currency string  "VND, USD"  
		image_url string  "Thumbnail Agoda"  
		rating float  "Số sao"  
		external_link string  "Link affiliate booking"  
	}

	ACTIVITY_VOTES {
		activity_id int PK,FK ""  
		user_id int PK,FK ""  
		vote_type string  ""  
	}

	EXPENSES {
		id int PK ""  
		group_id int FK ""  
		payer_id int FK ""  
		amount decimal  ""  
		currency string  ""  
		description string  ""  
		split_type string  ""  
	}

	EXPENSE_SPLITS {
		expense_id int PK,FK ""  
		user_id int PK,FK ""  
		amount_owed decimal  ""  
	}

	CHECKLIST_ITEMS {
		id int PK ""  
		group_id int FK ""  
		title string  ""  
		category string  ""  
		assignee_id int FK ""  
		is_completed boolean  ""  
		completed_by_id int  ""  
	}

	DOCUMENTS {
		id int PK ""  
		activity_id int FK ""  
		group_id int FK ""  
		uploaded_by_id int FK ""  
		file_url string  ""  
		file_type string  ""  
		file_name string  ""  
		file_size float  ""  
		category string  ""  
	}

	USERS||--o{GROUP_MEMBERS:"tham gia"
	USERS||--o{GROUPS:"tạo (owner)"
	USERS||--o{ACTIVITIES:"đề xuất"
	USERS||--o{ACTIVITY_VOTES:"bình chọn"
	USERS||--o{EXPENSES:"thanh toán (payer)"
	USERS||--o{EXPENSE_SPLITS:"nợ tiền (debtor)"
	USERS||--o{CHECKLIST_ITEMS:"được phân công"
    USERS ||--o{ DOCUMENTS : "tải lên"
	GROUPS||--o{GROUP_MEMBERS:"chứa"
	GROUPS||--o{ACTIVITIES:"thuộc về"
	GROUPS||--o{EXPENSES:"chi tiêu của"
	GROUPS||--o{DOCUMENTS:"tài liệu của"
	GROUPS||--o{CHECKLIST_ITEMS:"công việc của"
	ACTIVITIES||--o{ACTIVITY_VOTES:"nhận"
	ACTIVITIES||--o{DOCUMENTS:"có tài liệu đính kèm"
	EXPENSES||--o{EXPENSE_SPLITS:"được chia nhỏ"
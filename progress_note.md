Báo cáo đánh giá DB & Danh sách công việc TripSyncV2
Sau khi đọc file 
TripSyncV2.md
 và kiểm tra lại source code (Models, UseCases, Controllers, DTOs), mình gửi bạn báo cáo tình hình các case cũ và danh sách các việc cần làm tiếp theo.

1. Đánh giá các case cũ (Sau khi sửa DB)
Việc cập nhật các bảng GROUPS và ACTIVITIES có ảnh hưởng đến các API hiện tại. Dưới đây là chi tiết:

WARNING

API Tạo Hoạt Động (Manual Activity Creation) - ĐANG LỖI (Break Contract) Trong file 
dto/activityDTO.go
, struct 
CreateActivityReq
 đã yêu cầu bắt buộc (binding:"required") đối với 2 trường mới:

Type (HOTEL / ATTRACTION / RESTAURANT / CAMPING)
Location Hệ quả: Nếu hiện tại frontend (code cũ) gọi API tạo hoạt động mà không truyền type hoặc location, API sẽ lập tức văng lỗi 400 Bad Request ("Dữ liệu đầu vào không hợp lệ").
NOTE

API Tạo Nhóm (Create Group) - VẪN CHẠY NHƯNG CHƯA HOÀN THIỆN Struct 
CreateGroupRequest
 (trong 
controllers/group_controller.go
) hiện chỉ nhận: Name, Description, StartDate, EndDate. Hệ quả: API này vẫn hoạt động bình thường (không bị crash) vì GORM chỉ insert rỗng/0 vào các trường mới (RouteDestinations, AccommodationPref, ExpectedMembers, BudgetPerPerson, Currency). Tuy nhiên, cần phải nâng cấp API này để nhận các tham số mới nhằm phục vụ tính năng gọi AI tạo lịch trình.

2. Danh sách công việc phát sinh (Bổ sung AI & Agoda)
Đây là các công việc bạn cần làm ngày để hoàn thiện luồng nghiệp vụ AI mới:

- Cập nhật API Create Group (Tạo Nhóm):
Sửa lại 
CreateGroupRequest
 DTO để frontend có thể truyền lên RouteDestinations, AccommodationPref, ExpectedMembers, BudgetPerPerson, Currency.
- Viết Service Tích hợp Gemini API (AI Planning):
Viết hàm gọi HTTP/SDK sang Gemini.
Xây dựng Prompt Engineering truyền các thông số của nhóm vào để Gemini trả về một danh sách hoạt động dưới dạng JSON (bao gồm tọa độ dự kiến, chi phí, tên điểm đến).
- Viết Service Tích hợp Agoda API:
Viết hàm gọi hoặc crawl dữ liệu từ Agoda dựa theo struct JSON đã định nghĩa ở phần 6.
Nhận vào địa điểm và ngân sách, trả về 3-5 Struct AgodaHotel.
- Xây dựng Hybrid Route (Usecase Tạo Lịch Trình Tự Động):
Logic: Tạo nhóm xong -> Gọi Gemini đề xuất -> Nếu đề xuất có ngủ (HOTEL/MIXED) thì gọi thêm Agoda lấy khách sạn thật -> Insert tất cả vào bảng ACTIVITIES với status='PENDING' và is_ai_generated=true.
- Cập nhật API Get Group Detail:
Đảm bảo trả về đầy đủ các trường cấu hình mới của Group để UI có thể hiển thị (ví dụ: Ngân sách, Hành trình).

3. Các API còn lại cần phải làm (Theo TripSyncV2.md)
Dựa trên Blueprint, đây là các Module và API hoàn toàn mới cần được xây dựng bổ sung từ đầu:

3.1. Module Quản lý Chi Tiêu (Smart Bill Splitter)
CRUD Expenses: Thêm, sửa, xóa khoản chi tiêu (EXPENSES).
Chia tiền (Split Logic): Thuật toán và API ghi nhận dữ liệu vào bảng EXPENSE_SPLITS (chia đều, chia theo tỷ lệ, hoặc tick chọn từng người).
Tính toán công nợ: API trả về bảng tóm tắt "Ai nợ ai bao nhiêu tiền".
3.2. Module Chia sẻ Tài Liệu (Document Vault)
Upload File: API tải tệp (vé máy bay, booking khách sạn) lên Server/Cloud (S3/Cloudinary) và lưu vào bảng DOCUMENTS.
Download/Get Datalist: Lấy danh sách tài liệu của một hoạt động hoặc toàn nhóm.
3.3. Module Checklist & Task
CRUD Checklist: API tạo, sửa, xóa các item cần mang/chuẩn bị (CHECKLIST_ITEMS).
Phân công (Assign): Gắn assignee_id cho thành viên và Toggle trạng thái hoàn thành (is_completed).
3.4. Module System Admin (Tùy chọn)
Quản trị viên hệ thống: Các API Get All Users, Get All Groups, Thống kê hệ thống dành cho Role Admin tổng.


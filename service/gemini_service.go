package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"tripsync-backend/models"
	"tripsync-backend/repository"
)

// GeminiActivity là struct hứng dữ liệu JSON chính xác từ Prompt trả về
type GeminiActivity struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"` // HOTEL, ATTRACTION, RESTAURANT, CAMPING
	Location      string  `json:"location"`
	Description   string  `json:"description"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	EstimatedCost float64 `json:"estimated_cost"`
	Lat           float64 `json:"lat"` // Bổ sung Tọa độ theo yêu cầu của bạn
	Lng           float64 `json:"lng"`
}

type GeminiService interface {
	GenerateItinerary(ctx context.Context, group *models.Group) ([]GeminiActivity, error)
}

type geminiServiceImpl struct {
	apiKey  string
	actRepo repository.ActivityRepository
}

func NewGeminiService(actRepo repository.ActivityRepository) GeminiService {
	// Lấy key từ biến môi trường, nhớ khai báo trong file .env nhé!
	return &geminiServiceImpl{
		apiKey:  os.Getenv("GEMINI_API_KEY"),
		actRepo: actRepo,
	}
}

func (s *geminiServiceImpl) GenerateItinerary(ctx context.Context, group *models.Group) ([]GeminiActivity, error) {
	// 1. PROMPT ENGINEERING
	// Truyền toàn bộ ngữ cảnh của nhóm vào đây để ép AI lên lịch trình chuẩn
	totalGroupBudget := group.BudgetPerPerson * float64(group.ExpectedMembers)

	prompt := fmt.Sprintf(`Bạn là một chuyên gia lên kế hoạch du lịch cực kỳ logic. Hãy tạo một lịch trình chi tiết dựa trên thông tin sau:
- Điểm xuất phát: %s
- Các điểm đến: %s
- Hành trình đầy đủ: %s → %s
- Thời gian: Từ %s đến %s
- Quy mô đoàn: %d người
- Ngân sách trung bình cho 1 người: %.0f %s
- TỔNG NGÂN SÁCH CẢ ĐOÀN (%d người): %.0f %s

YÊU CẦU BẮT BUỘC:
1. Phân bổ ngân sách hợp lý cho ăn uống, vui chơi và lưu trú dựa trên con số TỔNG %.0f %s.
2. Với các hoạt động lưu trú (type: HOTEL), trường 'location' CHỈ GHI TÊN THÀNH PHỐ.
3. Trường 'estimated_cost' TRONG JSON PHẢI LÀ TỔNG CHI PHÍ CHO CẢ ĐOÀN %d NGƯỜI (Ví dụ: Tiền thuê đủ phòng cho %d người, tiền ăn cho cả đoàn). KHÔNG ĐƯỢC để giá cho 1 người.
4. Ước lượng tọa độ (lat, lng) của điểm đến.
5. Trả về DUY NHẤT một mảng JSON array.
6. Lên lịch trình hợp lý cho từng ngày. PHẢI ĐẢM BẢO NGÀY NÀO CŨNG CÓ LỊCH TRÌNH từ ngày %s đến ngày %s.
7. Nếu ngân sách đã cạn kiệt ở những ngày cuối, HÃY ĐỀ XUẤT CÁC HOẠT ĐỘNG MIỄN PHÍ (như dạo biển, ngắm hoàng hôn, tự do khám phá thành phố) để lấp đầy các ngày đó. Tuyệt đối không để trống lịch trình.
8. Múi giờ của 'start_time' và 'end_time' PHẢI LÀ giờ Việt Nam (UTC+7), định dạng "YYYY-MM-DDTHH:mm:ss+07:00". Dữ liệu không được dùng múi giờ Z (UTC).
9. Trường 'type' BẮT BUỘC phải là một trong 5 giá trị sau, KHÔNG ĐƯỢC để trống hoặc dùng giá trị khác: "HOTEL", "ATTRACTION", "RESTAURANT", "CAMPING", "TRANSPORT". Quy tắc: lưu trú → HOTEL, ăn uống/quán ăn/nhà hàng → RESTAURANT, tham quan/vui chơi/hoạt động ngoài trời → ATTRACTION, cắm trại → CAMPING, di chuyển/phương tiện/xe/tàu/máy bay → TRANSPORT.
10. BẮT BUỘC phải có ít nhất 1 hoạt động type TRANSPORT vào ngày đầu tiên (%s) để mô tả hành trình di chuyển từ điểm xuất phát "%s" đến điểm đến đầu tiên. Ước tính thời gian và chi phí di chuyển thực tế (xe khách, tàu hỏa, máy bay tùy khoảng cách). Nếu hành trình có nhiều chặng, thêm TRANSPORT cho mỗi chặng di chuyển giữa các điểm đến.

ĐỊNH DẠNG JSON MẪU:
[
  {
    "name": "Khách sạn trung tâm",
    "type": "HOTEL",
    "location": "Hà Nội",
    "description": "Nơi ở cho cả đoàn 8 người",
    "start_time": "YYYY-MM-DDTHH:mm:ss+07:00",
    "end_time": "YYYY-MM-DDTHH:mm:ss+07:00",
    "estimated_cost": 4000000, // Ví dụ: Tổng tiền 4 phòng cho 8 người
    "lat": 21.028511,
    "lng": 105.804817
  }
]`,
		group.DepartureLocation,
		group.RouteDestinations,
		group.DepartureLocation, group.RouteDestinations,
		group.StartDate.Format("2006-01-02"),
		group.EndDate.Format("2006-01-02"),
		group.ExpectedMembers,
		group.BudgetPerPerson, group.Currency,
		group.ExpectedMembers, totalGroupBudget, group.Currency, // Nhấn mạnh con số tổng
		totalGroupBudget, group.Currency,
		group.ExpectedMembers, group.ExpectedMembers,
		group.StartDate.Format("2006-01-02"),
		group.EndDate.Format("2006-01-02"),
		group.StartDate.Format("2006-01-02"), group.DepartureLocation)

	// Append rating context section if available
	ratingSection := s.buildRatingContextSection(ctx)
	if ratingSection != "" {
		prompt += "\n\n" + ratingSection
	}

	// 2. Build Payload (Sử dụng model gemini-1.5-flash cho tốc độ phản hồi nhanh)
	requestBody, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json", // Ép Gemini trả về JSON thuần túy
		},
	})

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=%s", s.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("lỗi khởi tạo request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 3. Gọi HTTP Request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gọi Gemini API: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	// Kiểm tra status code (Thường là 200 OK)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini API trả về lỗi: %s", string(bodyBytes))
	}

	// 4. Parse JSON Response từ cấu trúc của Google
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("không thể parse response từ Google: %v", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini không trả về kết quả nào")
	}

	// 5. Unmarshal text JSON của AI thành struct Go của mình
	jsonText := geminiResp.Candidates[0].Content.Parts[0].Text
	var activities []GeminiActivity
	if err := json.Unmarshal([]byte(jsonText), &activities); err != nil {
		return nil, fmt.Errorf("lỗi khi map JSON của AI vào struct: %v \nNội dung AI trả về: %s", err, jsonText)
	}

	return activities, nil
}

func (s *geminiServiceImpl) buildRatingContextSection(ctx context.Context) string {
	if s.actRepo == nil {
		return ""
	}
	ratingCtx, err := s.actRepo.GetRatingContext(ctx)
	if err != nil {
		fmt.Printf("⚠️ Không thể lấy rating context: %v\n", err)
		return ""
	}
	if len(ratingCtx.HighlyRated) == 0 && len(ratingCtx.PoorlyRated) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("KINH NGHIỆM TỪ CÁC CHUYẾN ĐI TRƯỚC (dựa trên đánh giá của người dùng):\n")

	if len(ratingCtx.HighlyRated) > 0 {
		sb.WriteString("Các hoạt động được đánh giá cao (nên tham khảo):\n")
		for _, item := range ratingCtx.HighlyRated {
			sb.WriteString(fmt.Sprintf("- %s (type: %s, địa điểm: %s) - ⭐ %.1f/5\n",
				item.Name, item.Type, item.Location, item.AverageUserRating))
		}
	}

	if len(ratingCtx.PoorlyRated) > 0 {
		sb.WriteString("Các hoạt động bị đánh giá thấp (nên tránh):\n")
		for _, item := range ratingCtx.PoorlyRated {
			sb.WriteString(fmt.Sprintf("- %s (type: %s, địa điểm: %s) - ⭐ %.1f/5\n",
				item.Name, item.Type, item.Location, item.AverageUserRating))
		}
	}

	return sb.String()
}

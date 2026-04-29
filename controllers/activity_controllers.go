package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ActivityController struct {
	useCase usecase.ActivityUseCase
}

func NewActivityController(u usecase.ActivityUseCase) *ActivityController {
	return &ActivityController{useCase: u}
}

func (c *ActivityController) GetActivities(ctx *gin.Context) {
	// 1. Lấy group_id từ URL
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	// 2. Lấy user_id từ context (do Middleware Auth set vào)
	userIDVal, _ := ctx.Get("user_id")
	userID := int(userIDVal.(float64))

	// 3. Gọi UseCase
	activities, err := c.useCase.GetGroupActivities(ctx.Request.Context(), groupID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Trả về Response
	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    activities,
	})
}

func (c *ActivityController) CreateActivity(ctx *gin.Context) {
	// 1. Lấy group_id từ URL (ví dụ: /groups/1/activities)
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	// 2. Lấy user_id từ Context (Giả sử Middleware Auth của bạn đã set "user_id")
	// Chỗ này tùy cách bạn viết Middleware, thường sẽ lưu float64 (nếu dùng JWT parse ra) hoặc string
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64)) // Ép kiểu cẩn thận tùy theo token của bạn nhé

	// 3. Hứng và Validate Payload từ Client
	var req dto.CreateActivityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Dữ liệu đầu vào không hợp lệ",
			"details": err.Error(),
		})
		return
	}

	err = c.useCase.CreateActivity(ctx.Request.Context(), groupID, userID, req)
	if err != nil {
		// Ở đây nếu xịn hơn có thể check err là NotFound hay InternalServerError để trả status code cho chuẩn
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 5. Trả về thành công
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Thêm hoạt động thành công!",
	})
}

func (c *ActivityController) ToggleVote(ctx *gin.Context) {
	// 1. Lấy activity_id từ URL
	activityIDStr := ctx.Param("activity_id")
	activityID, err := strconv.Atoi(activityIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID hoạt động không hợp lệ"})
		return
	}

	// 2. Lấy user_id từ token (đã qua Auth Middleware)
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64)) // Ép kiểu tùy setup JWT của bạn

	// 3. Gọi UseCase
	isAdded, err := c.useCase.ToggleActivityVote(ctx.Request.Context(), activityID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi xử lý bình chọn"})
		return
	}

	// 4. Trả về Response
	msg := "Đã bỏ bình chọn"
	if isAdded {
		msg = "Đã bình chọn thành công"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  msg,
		"is_added": isAdded, // Frontend dùng cờ này để update lại state UI (xanh lá/xám)
	})
}

func (c *ActivityController) FinalizeActivity(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)
	activityIDStr := ctx.Param("activity_id")
	activityID, _ := strconv.Atoi(activityIDStr)
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))
	err := c.useCase.FinalizeActivity(ctx.Request.Context(), uint(groupID), uint(activityID), uint(userID))
	if err != nil {
		if err.Error() == "chỉ Admin mới có quyền chốt hoạt động vào lịch chính thức" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Đã chốt hoạt động vào lịch chính thức", "status": "APPROVED"})
}

func (c *ActivityController) UnfinalizeActivity(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)
	activityIDStr := ctx.Param("activity_id")
	activityID, _ := strconv.Atoi(activityIDStr)
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))
	err := c.useCase.UnfinalizeActivity(ctx.Request.Context(), uint(groupID), uint(activityID), uint(userID))
	if err != nil {
		if err.Error() == "chỉ Admin mới có quyền hủy chốt hoạt động" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Đã hủy chốt hoạt động", "status": "PENDING"})
}

func (c *ActivityController) UpdateActivity(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id")) // param :id của group
	activityID, _ := strconv.Atoi(ctx.Param("activity_id"))
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	var req dto.UpdateActivityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if err := c.useCase.UpdateActivity(userID, groupID, activityID, req); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật hoạt động thành công",
	})
}

func (c *ActivityController) DeleteActivity(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id"))
	activityID, _ := strconv.Atoi(ctx.Param("activity_id"))
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	if err := c.useCase.DeleteActivity(userID, groupID, activityID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Xóa hoạt động thành công",
	})
}

func (c *ActivityController) RateActivity(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	activityIDStr := ctx.Param("activity_id")
	activityID, err := strconv.Atoi(activityIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	var req dto.RateActivityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Rating must be between 1 and 5"})
		return
	}

	if err := c.useCase.RateActivity(ctx.Request.Context(), groupID, activityID, userID, req.Rating); err != nil {
		if err.Error() == "bạn không phải là thành viên của nhóm này" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đánh giá thành công"})
}

func (c *ActivityController) GetSuggestions(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	activityType := ctx.Query("type")
	location := ctx.Query("location")

	suggestions, err := c.useCase.GetSuggestions(ctx.Request.Context(), groupID, activityType, location)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "type must be one of HOTEL, ATTRACTION, RESTAURANT, CAMPING" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		if errMsg == "location query parameter is required" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success", "data": suggestions})
}

func (c *ActivityController) DeleteAllActivities(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id"))
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	if err := c.useCase.DeleteAllActivities(ctx.Request.Context(), groupID, userID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đã xóa toàn bộ lịch trình"})
}

// ExportActivities handles GET /api/groups/:id/export (public, no auth required).
// Returns the group's activities as an export payload without internal fields.
func (c *ActivityController) ExportActivities(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	items, err := c.useCase.ExportActivities(ctx.Request.Context(), groupID)
	if err != nil {
		if err.Error() == "group_not_found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Nhóm không tồn tại"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		return
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := "Sheet1"
	headers := []string{"STT", "Tên hoạt động", "Loại", "Địa điểm", "Ngày", "Giờ bắt đầu", "Giờ kết thúc", "Chi phí ước tính", "Tiền tệ", "Trạng thái", "Mô tả", "Lat", "Lng"}
	
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"3E7336"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	
	for i, header := range headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		cell := colName + "1"
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	
	// Hide Lat, Lng
	f.SetColVisible(sheetName, "L", false)
	f.SetColVisible(sheetName, "M", false)

	for i, item := range items {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), item.Type)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), item.Location)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), item.StartTime.Format("02/01/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.StartTime.Format("15:04"))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), item.EndTime.Format("15:04"))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), item.EstimatedCost)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), item.Currency)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), "Đề xuất")
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), item.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), item.Lat)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), item.Lng)
	}

	f.SetColWidth(sheetName, "A", "A", 5)
	f.SetColWidth(sheetName, "B", "B", 30)
	f.SetColWidth(sheetName, "C", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 25)
	f.SetColWidth(sheetName, "E", "G", 15)
	f.SetColWidth(sheetName, "H", "H", 15)
	f.SetColWidth(sheetName, "I", "J", 10)
	f.SetColWidth(sheetName, "K", "K", 40)

	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="group_%d_itinerary.xlsx"`, groupID))
	
	if err := f.Write(ctx.Writer); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi ghi file excel"})
	}
}

// ImportActivities handles POST /api/groups/:id/import (protected, auth required).
// Copies activities from source group into the target group (:id), remapping times to target start_date.
func (c *ActivityController) ImportActivities(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	targetGroupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	var req dto.ImportActivitiesReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	count, err := c.useCase.ImportActivities(ctx.Request.Context(), targetGroupID, req.SourceGroupID, userID)
	if err != nil {
		switch err.Error() {
		case "self_import":
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không thể import từ chính nhóm này"})
		case "not_admin_of_target":
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải Admin của nhóm đích"})
		case "not_member_of_source":
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm nguồn"})
		case "target_group_not_found":
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Nhóm đích không tồn tại"})
		case "exceeds_duration":
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Lịch trình import vượt quá số ngày của chuyến đi hiện tại"})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":        "Import thành công",
		"imported_count": count,
	})
}

// ImportFromExcel handles POST /api/groups/:id/import-excel (protected).
func (c *ActivityController) ImportFromExcel(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	targetGroupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file upload"})
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File không đúng định dạng Excel"})
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := f.GetSheetName(0) // Lấy sheet đầu tiên
	rows, err := f.GetRows(sheetName)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đọc dữ liệu từ sheet"})
		return
	}

	var activities []dto.ExportActivityItem
	for i, row := range rows {
		if i == 0 {
			continue // Bỏ qua dòng header
		}
		
		getCol := func(index int) string {
			if index < len(row) {
				return row[index]
			}
			return ""
		}

		name := getCol(1)
		if name == "" {
			continue // Bỏ qua nếu tên hoạt động trống
		}

		activityType := getCol(2)
		location := getCol(3)
		dateStr := getCol(4)      // DD/MM/YYYY
		startTimeStr := getCol(5) // HH:mm hoac giong vay
		endTimeStr := getCol(6)   // HH:mm
		costStr := getCol(7)
		currency := getCol(8)
		desc := getCol(10)
		latStr := getCol(11)
		lngStr := getCol(12)

		loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		if loc == nil {
			loc = time.Local
		}
		
		date, err := time.ParseInLocation("02/01/2006", dateStr, loc)
		if err != nil {
			continue
		}
		
		formatTime := func(t string) string {
			parts := strings.Split(t, ":")
			if len(parts) >= 2 {
				return parts[0] + ":" + parts[1]
			}
			return t
		}
		
		startTimeStr = formatTime(startTimeStr)
		endTimeStr = formatTime(endTimeStr)

		startTime, err := time.ParseInLocation("15:04", startTimeStr, loc)
		if err != nil {
			continue
		}
		startDateTime := time.Date(date.Year(), date.Month(), date.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
		
		endTime, err := time.ParseInLocation("15:04", endTimeStr, loc)
		if err != nil {
			continue
		}
		endDateTime := time.Date(date.Year(), date.Month(), date.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)
		if endDateTime.Before(startDateTime) {
			endDateTime = endDateTime.AddDate(0, 0, 1)
		}

		cost, _ := strconv.ParseFloat(costStr, 64)
		lat, _ := strconv.ParseFloat(latStr, 64)
		lng, _ := strconv.ParseFloat(lngStr, 64)

		activities = append(activities, dto.ExportActivityItem{
			Name:          name,
			Type:          activityType,
			Location:      location,
			Description:   desc,
			StartTime:     startDateTime,
			EndTime:       endDateTime,
			EstimatedCost: cost,
			Currency:      currency,
			Lat:           lat,
			Lng:           lng,
		})
	}

	count, err := c.useCase.ImportFromJSON(ctx.Request.Context(), targetGroupID, userID, activities)
	if err != nil {
		switch err.Error() {
		case "not_admin_of_target":
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải Admin của nhóm đích"})
		case "target_group_not_found":
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Nhóm đích không tồn tại"})
		case "exceeds_duration":
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Lịch trình import vượt quá số ngày của chuyến đi hiện tại"})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":        "Import từ file thành công",
		"imported_count": count,
	})
}

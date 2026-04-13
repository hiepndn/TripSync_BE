package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"tripsync-backend/models"
)

// --- STRUCTS CHO BƯỚC 1 (AUTO-COMPLETE) ---
type AgodaAutoCompleteResponse struct {
	Places []struct {
		Id     int `json:"id"`
		TypeId int `json:"typeId"` // <-- THÊM DÒNG NÀY
	} `json:"places"`
}

// --- STRUCTS CHO BƯỚC 2 (SEARCH) ĐÃ ĐƯỢC CHỐT TỪ TRƯỚC ---
type AgodaResponse struct {
	Data struct {
		CitySearch struct {
			Properties []AgodaHotel `json:"properties"`
		} `json:"citySearch"`
	} `json:"data"`
}

type AgodaHotel struct {
	PropertyResultType string `json:"propertyResultType"`
	Content            struct {
		InformationSummary struct {
			LocaleName string  `json:"localeName"`
			Rating     float64 `json:"rating"`
		} `json:"informationSummary"`
		GeoInfo struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"geoInfo"`
		Images struct {
			HotelImages []struct {
				Urls []struct {
					Value string `json:"value"`
				} `json:"urls"`
			} `json:"hotelImages"`
		} `json:"images"`
	} `json:"content"`

	Pricing struct {
		Offers []struct {
			RoomOffers []struct {
				Room struct {
					Pricing []struct {
						Currency string `json:"currency"`
						Price    struct {
							PerRoomPerNight struct {
								Inclusive struct {
									Display float64 `json:"display"`
								} `json:"inclusive"`
							} `json:"perRoomPerNight"`
						} `json:"price"`
					} `json:"pricing"`
				} `json:"room"`
			} `json:"roomOffers"`
		} `json:"offers"`
	} `json:"pricing"`
}

type AgodaService interface {
	SearchHotels(ctx context.Context, cityName string, checkIn string, checkOut string, budget float64, currency string, expectedMembers int, groupID uint) ([]models.Activity, error)
}

type agodaServiceImpl struct {
	rapidApiKey  string
	rapidApiHost string
	client       *http.Client
}

func NewAgodaService() AgodaService {
	return &agodaServiceImpl{
		rapidApiKey:  os.Getenv("RAPIDAPI_KEY"),
		rapidApiHost: "agoda-com.p.rapidapi.com",
		client:       &http.Client{},
	}
}

// Hàm phụ trợ để set Header cho RapidAPI
func (s *agodaServiceImpl) setHeaders(req *http.Request) {
	req.Header.Add("X-RapidAPI-Key", s.rapidApiKey)
	req.Header.Add("X-RapidAPI-Host", s.rapidApiHost)
}

func (s *agodaServiceImpl) SearchHotels(ctx context.Context, cityName string, checkIn string, checkOut string, budget float64, currency string, expectedMembers int, groupID uint) ([]models.Activity, error) {
	// =========================================================================
	// 🌟 BƯỚC 1: GỌI AUTO-COMPLETE (SỬA LẠI CÁCH ENCODE %20)
	// =========================================================================
	// Thay vì dùng url.QueryEscape sinh ra dấu "+", ta ép nó dùng "%20"
	encodedCity := strings.ReplaceAll(cityName, " ", "%20")
	autoCompleteUrl := fmt.Sprintf("https://agoda-com.p.rapidapi.com/hotels/auto-complete?query=%s", encodedCity)

	req1, _ := http.NewRequestWithContext(ctx, "GET", autoCompleteUrl, nil)
	s.setHeaders(req1)

	resp1, err := s.client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("lỗi request auto-complete: %v", err)
	}
	defer resp1.Body.Close()

	body1, _ := io.ReadAll(resp1.Body)

	var acData AgodaAutoCompleteResponse
	if err := json.Unmarshal(body1, &acData); err != nil {
		fmt.Printf("❌ [DEBUG] Lỗi Parse AutoComplete. Dữ liệu thô từ API: %s\n", string(body1))
		return nil, fmt.Errorf("lỗi parse auto-complete: %v", err)
	}

	placeId := ""
	if len(acData.Places) > 0 {
		// Nối typeId và id lại với nhau thành định dạng "1_2758"
		placeId = fmt.Sprintf("%d_%d", acData.Places[0].TypeId, acData.Places[0].Id)
	}

	if placeId == "" {
		fmt.Printf("❌ [DEBUG] AutoComplete không tìm thấy ID cho '%s'. Dữ liệu trả về: %s\n", cityName, string(body1))
		return nil, fmt.Errorf("không tìm thấy placeId cho địa điểm")
	}

	// =========================================================================
	// 🌟 BƯỚC 2: TÌM KHÁCH SẠN
	// =========================================================================
	numRooms := (expectedMembers + 1) / 2
	adults := expectedMembers
	searchUrl := fmt.Sprintf("https://agoda-com.p.rapidapi.com/hotels/search-overnight?id=%s&checkinDate=%s&checkoutDate=%s&currency=%s&rooms=%d&adults=%d",
		placeId, checkIn, checkOut, currency, numRooms, adults)

	req2, _ := http.NewRequestWithContext(ctx, "GET", searchUrl, nil)
	s.setHeaders(req2)

	resp2, err := s.client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("lỗi request search-overnight: %v", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)

	var agodaData AgodaResponse
	if err := json.Unmarshal(body2, &agodaData); err != nil {
		// 🌟 SỬA ĐOẠN NÀY ĐỂ TRÁNH PANIC
		limit := len(body2)
		if limit > 200 {
			limit = 200
		}
		fmt.Printf("❌ [DEBUG] Lỗi Parse Search. Dữ liệu thô (%d ký tự): %s\n", len(body2), string(body2)[:limit])
		return nil, fmt.Errorf("lỗi parse search-overnight: %v", err)
	}

	// =========================================================================
	// 🌟 BƯỚC 3: MAP SANG MODELS
	// =========================================================================
	var hotelActivities []models.Activity
	limit := 3
	count := 0

	for _, prop := range agodaData.Data.CitySearch.Properties {
		if count >= limit {
			break
		}

		if prop.PropertyResultType == "SoldOutProperty" || len(prop.Pricing.Offers) == 0 || len(prop.Pricing.Offers[0].RoomOffers) == 0 {
			continue
		}

		pricingArr := prop.Pricing.Offers[0].RoomOffers[0].Room.Pricing
		if len(pricingArr) == 0 {
			continue
		}

		actualPrice := pricingArr[0].Price.PerRoomPerNight.Inclusive.Display
		resCurrency := pricingArr[0].Currency

		//Nới lỏng dung sai budget lên 1.5 lần (Tránh trường hợp Agoda toàn phòng đắt khiến mảng bị rỗng)
		if actualPrice > budget*1.2 {
			continue
		}

		imgUrl := ""
		if len(prop.Content.Images.HotelImages) > 0 && len(prop.Content.Images.HotelImages[0].Urls) > 0 {
			imgUrl = "https:" + prop.Content.Images.HotelImages[0].Urls[0].Value
		}

		newHotel := models.Activity{
			GroupID:       groupID,
			Name:          prop.Content.InformationSummary.LocaleName,
			Type:          "HOTEL",
			Location:      cityName,
			Description:   fmt.Sprintf("Khách sạn tự động chọn từ Agoda. Đánh giá: %.1f/10", prop.Content.InformationSummary.Rating),
			Status:        models.StatusPending,
			CreatedBy:     nil,
			Lat:           prop.Content.GeoInfo.Latitude,
			Lng:           prop.Content.GeoInfo.Longitude,
			IsAIGenerated: true,
			EstimatedCost: actualPrice,
			Currency:      resCurrency,
			ImageURL:      imgUrl,
			Rating:        prop.Content.InformationSummary.Rating,
		}

		hotelActivities = append(hotelActivities, newHotel)
		count++
	}

	// Cảnh báo nếu Agoda có trả về list khách sạn nhưng bị loại hết bởi bộ lọc giá
	if len(hotelActivities) == 0 {
		fmt.Printf("⚠️ [DEBUG] Agoda có dữ liệu nhưng không có KS nào khớp với budget (%.0f %s) tại %s\n", budget, currency, cityName)
	}

	return hotelActivities, nil
}

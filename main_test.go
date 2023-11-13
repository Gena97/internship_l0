package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrderHandler(t *testing.T) {
	// Создаем фейковый заказ для тестирования
	fakeOrderID := "fakeOrderID"
	fakeOrder := Order{
		OrderUID:    "fakeOrderID",
		TrackNumber: "123456789",
		Entry:       "Entry1",
		Delivery: struct {
			Name    string `json:"name"`
			Phone   string `json:"phone"`
			Zip     string `json:"zip"`
			City    string `json:"city"`
			Address string `json:"address"`
			Region  string `json:"region"`
			Email   string `json:"email"`
		}{
			Name:    "John Doe",
			Phone:   "1234567890",
			Zip:     "12345",
			City:    "Cityville",
			Address: "123 Main St",
			Region:  "Region1",
			Email:   "john.doe@example.com",
		},
		Payment: struct {
			Transaction  string  `json:"transaction"`
			RequestID    string  `json:"request_id"`
			Currency     string  `json:"currency"`
			Provider     string  `json:"provider"`
			Amount       float64 `json:"amount"`
			PaymentDT    int64   `json:"payment_dt"`
			Bank         string  `json:"bank"`
			DeliveryCost float64 `json:"delivery_cost"`
			GoodsTotal   float64 `json:"goods_total"`
			CustomFee    float64 `json:"custom_fee"`
		}{
			Transaction:  "trans123",
			RequestID:    "req123",
			Currency:     "USD",
			Provider:     "Provider1",
			Amount:       100.00,
			PaymentDT:    1636600000,
			Bank:         "Bank1",
			DeliveryCost: 10.00,
			GoodsTotal:   90.00,
			CustomFee:    5.00,
		},
		Items: []struct {
			ChrtID      int     `json:"chrt_id"`
			TrackNumber string  `json:"track_number"`
			Price       float64 `json:"price"`
			RID         string  `json:"rid"`
			Name        string  `json:"name"`
			Sale        int     `json:"sale"`
			Size        string  `json:"size"`
			TotalPrice  float64 `json:"total_price"`
			NmID        int     `json:"nm_id"`
			Brand       string  `json:"brand"`
			Status      int     `json:"status"`
		}{
			{
				ChrtID:      1,
				TrackNumber: "item123",
				Price:       30.00,
				RID:         "rid123",
				Name:        "Item1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  30.00,
				NmID:        101,
				Brand:       "Brand1",
				Status:      1,
			},
		},
		Locale:            "en_US",
		InternalSignature: "internal_sign_123",
		CustomerID:        "customer123",
		DeliveryService:   "DeliveryService1",
		ShardKey:          "shard_key_1",
		SmID:              1,
		DateCreated:       "2023-11-10T12:00:00Z",
		OOFShard:          "oof_shard_1",
	}

	// Добавляем фейковый заказ в кеш
	cacheMutex.Lock()
	orderCache[fakeOrderID] = fakeOrder
	cacheMutex.Unlock()

	// Создаем HTTP-запрос с параметром id=fakeOrderID
	req, err := http.NewRequest("GET", "/order?id="+fakeOrderID, nil)
	if err != nil {
		t.Fatalf("Не удалось создать HTTP-запрос: %v", err)
	}

	// Создаем фейковый HTTP-ResponseWriter для записи ответа
	recorder := httptest.NewRecorder()

	// Вызываем обработчик HTTP-запроса
	getOrderHandler(recorder, req)

	// Проверяем код состояния ответа
	if recorder.Code != http.StatusOK {
		t.Errorf("Ожидался код состояния %d, получено: %d", http.StatusOK, recorder.Code)
	}

	// Декодируем ответ и сравниваем с ожидаемым результатом
	var responseOrder Order
	err = json.NewDecoder(recorder.Body).Decode(&responseOrder)
	if err != nil {
		t.Fatalf("Ошибка декодирования JSON-ответа: %v", err)
	}

	// Проверяем, что полученный заказ соответствует фейковому заказу
	if responseOrder.OrderUID != fakeOrder.OrderUID {
		t.Errorf("Ожидался OrderUID %s, получено: %s", fakeOrder.OrderUID, responseOrder.OrderUID)
	}

	// Очищаем фейковый заказ из кеша
	cacheMutex.Lock()
	delete(orderCache, fakeOrderID)
	cacheMutex.Unlock()
}

func TestGetOrderHandler_OrderNotFound(t *testing.T) {
	// Создаем HTTP-запрос с параметром id=nonexistentOrderID
	req, err := http.NewRequest("GET", "/order?id=nonexistentOrderID", nil)
	if err != nil {
		t.Fatalf("Не удалось создать HTTP-запрос: %v", err)
	}

	// Создаем фейковый HTTP-ResponseWriter для записи ответа
	recorder := httptest.NewRecorder()

	// Вызываем обработчик HTTP-запроса
	getOrderHandler(recorder, req)

	// Проверяем код состояния ответа
	if recorder.Code != http.StatusNotFound {
		t.Errorf("Ожидался код состояния %d, получено: %d", http.StatusNotFound, recorder.Code)
	}

	// Проверяем текст ответа
	expectedBody := "Заказ не найден"
	if body := recorder.Body.String(); body != expectedBody {
		t.Errorf("Ожидался текст %s, получено: %s", expectedBody, body)
	}
}

func TestGetOrderHandler_InvalidID(t *testing.T) {
	// Создаем HTTP-запрос с параметром id=invalidOrderID (некорректный формат ID)
	req, err := http.NewRequest("GET", "/order?id=invalidOrderID", nil)
	if err != nil {
		t.Fatalf("Не удалось создать HTTP-запрос: %v", err)
	}

	// Создаем фейковый HTTP-ResponseWriter для записи ответа
	recorder := httptest.NewRecorder()

	// Вызываем обработчик HTTP-запроса
	getOrderHandler(recorder, req)

	// Проверяем код состояния ответа
	if recorder.Code != http.StatusNotFound {
		t.Errorf("Ожидался код состояния %d, получено: %d", http.StatusNotFound, recorder.Code)
	}

	// Проверяем текст ответа
	expectedBody := "Заказ не найден"
	if body := recorder.Body.String(); body != expectedBody {
		t.Errorf("Ожидался текст %s, получено: %s", expectedBody, body)
	}
}

func TestHTTPServer(t *testing.T) {
	// Создаем фейковый заказ для тестирования
	fakeOrderID := "fakeOrderID"
	fakeOrder := Order{
		OrderUID:    "fakeOrderID",
		TrackNumber: "123456789",
		Entry:       "Entry1",
		Delivery: struct {
			Name    string `json:"name"`
			Phone   string `json:"phone"`
			Zip     string `json:"zip"`
			City    string `json:"city"`
			Address string `json:"address"`
			Region  string `json:"region"`
			Email   string `json:"email"`
		}{
			Name:    "John Doe",
			Phone:   "1234567890",
			Zip:     "12345",
			City:    "Cityville",
			Address: "123 Main St",
			Region:  "Region1",
			Email:   "john.doe@example.com",
		},
		Payment: struct {
			Transaction  string  `json:"transaction"`
			RequestID    string  `json:"request_id"`
			Currency     string  `json:"currency"`
			Provider     string  `json:"provider"`
			Amount       float64 `json:"amount"`
			PaymentDT    int64   `json:"payment_dt"`
			Bank         string  `json:"bank"`
			DeliveryCost float64 `json:"delivery_cost"`
			GoodsTotal   float64 `json:"goods_total"`
			CustomFee    float64 `json:"custom_fee"`
		}{
			Transaction:  "trans123",
			RequestID:    "req123",
			Currency:     "USD",
			Provider:     "Provider1",
			Amount:       100.00,
			PaymentDT:    1636600000,
			Bank:         "Bank1",
			DeliveryCost: 10.00,
			GoodsTotal:   90.00,
			CustomFee:    5.00,
		},
		Items: []struct {
			ChrtID      int     `json:"chrt_id"`
			TrackNumber string  `json:"track_number"`
			Price       float64 `json:"price"`
			RID         string  `json:"rid"`
			Name        string  `json:"name"`
			Sale        int     `json:"sale"`
			Size        string  `json:"size"`
			TotalPrice  float64 `json:"total_price"`
			NmID        int     `json:"nm_id"`
			Brand       string  `json:"brand"`
			Status      int     `json:"status"`
		}{
			{
				ChrtID:      1,
				TrackNumber: "item123",
				Price:       30.00,
				RID:         "rid123",
				Name:        "Item1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  30.00,
				NmID:        101,
				Brand:       "Brand1",
				Status:      1,
			},
			// Добавьте другие товары по аналогии
		},
		Locale:            "en_US",
		InternalSignature: "internal_sign_123",
		CustomerID:        "customer123",
		DeliveryService:   "DeliveryService1",
		ShardKey:          "shard_key_1",
		SmID:              1,
		DateCreated:       "2023-11-10T12:00:00Z",
		OOFShard:          "oof_shard_1",
	}

	// Добавляем фейковый заказ в кеш
	cacheMutex.Lock()
	orderCache[fakeOrderID] = fakeOrder
	cacheMutex.Unlock()

	// Создаем новый HTTP-сервер и передаем ему обработчик запросов
	ts := httptest.NewServer(http.HandlerFunc(getOrderHandler))
	defer ts.Close()

	// Отправляем GET-запрос на тестовый сервер
	resp, err := http.Get(ts.URL + "/order?id=" + fakeOrderID)
	if err != nil {
		t.Fatalf("Не удалось выполнить GET-запрос: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем код состояния ответа
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ожидался код состояния %d, получено: %d", http.StatusOK, resp.StatusCode)
	}

	// Декодируем ответ и сравниваем с ожидаемым результатом
	var responseOrder Order
	err = json.NewDecoder(resp.Body).Decode(&responseOrder)
	if err != nil {
		t.Fatalf("Ошибка декодирования JSON-ответа: %v", err)
	}

	// Проверяем, что полученный заказ соответствует фейковому заказу
	if responseOrder.OrderUID != fakeOrder.OrderUID {
		t.Errorf("Ожидался OrderUID %s, получено: %s", fakeOrder.OrderUID, responseOrder.OrderUID)
	}

	// Очищаем фейковый заказ из кеша
	cacheMutex.Lock()
	delete(orderCache, fakeOrderID)
	cacheMutex.Unlock()
}

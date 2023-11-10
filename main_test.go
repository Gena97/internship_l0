package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestSaveOrder(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database connection: %v", err)
	}
	defer db.Close()

	// Заменяем db.Exec на mock
	mock.ExpectExec("INSERT INTO orders").WithArgs("test_order_uid", "test_track_number").WillReturnResult(sqlmock.NewResult(1, 1))

	// Создаем тестовый заказ
	order := Order{
		OrderUID:    "test_order_uid",
		TrackNumber: "test_track_number",
	}

	// Вызываем функцию saveOrder
	err = saveOrder(order)

	// Проверяем, что запрос к базе данных был выполнен
	assert.NoError(t, mock.ExpectationsWereMet(), "Expectations were not met")

	// Проверяем, что ошибки нет
	assert.NoError(t, err, "Error saving order")
}

func TestGetOrderHandler(t *testing.T) {
	// Создаем тестовый заказ
	orderID := "test_order_uid"
	order := Order{
		OrderUID:    orderID,
		TrackNumber: "test_track_number",
	}

	// Добавляем заказ в кэш
	orderCache[orderID] = order

	// Создаем HTTP-запрос
	req, err := http.NewRequest("GET", "/order?id="+orderID, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Обрабатываем запрос с помощью хендлера
	handler := http.HandlerFunc(getOrderHandler)
	handler.ServeHTTP(rr, req)

	// Проверяем код статуса
	assert.Equal(t, http.StatusOK, rr.Code, "Handler returned wrong status code")

	// Проверяем, что ответ содержит правильные данные заказа
	var responseOrder Order
	err = json.Unmarshal(rr.Body.Bytes(), &responseOrder)
	assert.NoError(t, err, "Error unmarshalling response")
	assert.Equal(t, order.OrderUID, responseOrder.OrderUID, "Handler returned unexpected order data")
}

func TestStartHTTPServer(t *testing.T) {
	// Создаем HTTP-запрос
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Обрабатываем запрос с помощью хендлера
	handler := http.HandlerFunc(startHTTPServer)
	handler.ServeHTTP(rr, req)

	// Проверяем код статуса
	assert.Equal(t, http.StatusNotFound, rr.Code, "Handler returned wrong status code")
}

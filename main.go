package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jackc/pgx/v4"
	"github.com/nats-io/stan.go"
)

// Переменная для хранения соединения с базой данных
var db *pgx.Conn

// Мьютекс для обеспечения безопасности работы с базой данных
var dbMutex sync.Mutex

// Мьютекс для безопасной работы с кешем заказов
var cacheMutex sync.RWMutex

// Структура для представления данных о заказе
type Order struct {
	OrderUID    string `json:"order_uid"`
	TrackNumber string `json:"track_number"`
	Entry       string `json:"entry"`
	Delivery    struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Zip     string `json:"zip"`
		City    string `json:"city"`
		Address string `json:"address"`
		Region  string `json:"region"`
		Email   string `json:"email"`
	} `json:"delivery"`
	Payment struct {
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
	} `json:"payment"`
	Items []struct {
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
	} `json:"items"`
	Locale            string `json:"locale"`
	InternalSignature string `json:"internal_signature"`
	CustomerID        string `json:"customer_id"`
	DeliveryService   string `json:"delivery_service"`
	ShardKey          string `json:"shardkey"`
	SmID              int    `json:"sm_id"`
	DateCreated       string `json:"date_created"`
	OOFShard          string `json:"oof_shard"`
}

// Кеш для хранения заказов в памяти
var orderCache = make(map[string]Order)

// Константы для подключения к NATS
const (
	natsURL     = "nats://localhost:4222"
	clusterID   = "test-cluster"
	clientID    = "your_client_id_2"
	subject     = "your_subject"
	durableName = "durable-order-sub"
	dbUser      = "Admin_Gena"
	dbPassword  = "pass"
	dbName      = "orders_db"
	dbPort      = 5433
)

// Основная функция приложения
func main() {
	// Подключение к базе данных
	if err := connectToDB(); err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close(context.Background())

	// Восстановление данных из базы в кеш
	restoreCacheFromDB()
	// Запуск подписки на сообщения от NATS
	go subscribeToNATS()
	// Запуск функции отправки данных в отдельной горутине
	go sender()
	// Обработчик запросов по пути "/order"
	http.HandleFunc("/order", getOrderHandler)
	// Запуск HTTP-сервера
	go startHTTPServer()

	// Ожидание завершения работы приложения (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

// Функция подключения к базе данных
func connectToDB() error {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable port=%d", dbUser, dbPassword, dbName, dbPort)
	var err error
	db, err = pgx.Connect(context.Background(), connStr)

	return err
}

// Функция восстановления данных из базы в кеш
func restoreCacheFromDB() {
	rows, err := db.Query(context.Background(), "SELECT order_uid, track_number FROM orders")
	if err != nil {
		log.Printf("Ошибка запроса заказов из PostgreSQL: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.OrderUID, &order.TrackNumber); err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}
		orderCache[order.OrderUID] = order
	}
}

// Функция подписки на сообщения от NATS
func subscribeToNATS() {
	sc, err := stan.Connect(clusterID, clientID)
	if err != nil {
		log.Fatalf("Ошибка подключения к NATS: %v", err)
	}
	defer sc.Close()

	subscription, err := sc.Subscribe(subject, func(msg *stan.Msg) {
		var orderData Order
		if err := json.Unmarshal(msg.Data, &orderData); err != nil {
			log.Printf("Ошибка десериализации данных заказа: %v", err)
			return
		}

		// Проверка валидности JSON
		if !json.Valid(msg.Data) {
			log.Printf("Получен невалидный JSON: %s", msg.Data)
			return
		}

		// Сохранение данных в базе данных и кэше
		saveOrder(orderData)
		updateOrderCache(orderData)
	}, stan.DurableName(durableName))

	if err != nil {
		log.Fatalf("Ошибка установки подписки на NATS: %v", err)
	}
	defer subscription.Unsubscribe()

	// Бесконечный цикл для поддержания подписки
	select {}
}

// Функция обновления кеша заказов
func updateOrderCache(order Order) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	orderCache[order.OrderUID] = order
}

// Функция сохранения данных заказа в базу данных
func saveOrder(order Order) {
	// Заблокировать мьютекс перед началом транзакции
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Начать транзакцию
	tx, err := db.Begin(context.Background())
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return
	}

	// Вставка данных в таблицу orders
	_, err = tx.Exec(context.Background(), "INSERT INTO orders VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		order.OrderUID, order.TrackNumber, order.Entry, order.DeliveryService, order.Locale,
		order.InternalSignature, order.CustomerID, order.ShardKey, order.SmID, order.DateCreated, order.OOFShard)
	if err != nil {
		log.Printf("Ошибка сохранения заказа в PostgreSQL: %v", err)
		tx.Rollback(context.Background()) // Откатить транзакцию при ошибке
		return
	}

	// Вставка данных в таблицу delivery
	_, err = tx.Exec(context.Background(), "INSERT INTO delivery VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		log.Printf("Ошибка сохранения информации о доставке в PostgreSQL: %v", err)
		tx.Rollback(context.Background()) // Откатить транзакцию при ошибке
		return
	}

	// Вставка данных в таблицу payment
	_, err = tx.Exec(context.Background(), "INSERT INTO payment VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)

	if err != nil {
		log.Printf("Ошибка сохранения информации о платеже в PostgreSQL: %v", err)
		tx.Rollback(context.Background()) // Откатить транзакцию при ошибке
		return
	}

	// Вставка данных в таблицу items
	for _, item := range order.Items {
		_, err := tx.Exec(context.Background(), "INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)",
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID, item.Name, item.Sale, item.Size,
			item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			log.Printf("Ошибка сохранения информации о товаре в PostgreSQL: %v", err)
			tx.Rollback(context.Background()) // Откатить транзакцию при ошибке
			return
		}
	}

	// Подтвердить транзакцию, если все операции успешны
	err = tx.Commit(context.Background())
	if err != nil {
		log.Printf("Ошибка подтверждения транзакции: %v", err)
		tx.Rollback(context.Background()) // Откатить транзакцию при ошибке
		return
	}
}

// Обработчик HTTP-запросов для получения данных о заказе
func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("id")

	// Заблокировать мьютекс для чтения из кеша
	cacheMutex.RLock()
	order, ok := orderCache[orderID]
	cacheMutex.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Заказ не найден"))
		return
	}

	// Отправить данные заказа в ответ на HTTP-запрос
	json.NewEncoder(w).Encode(order)
}

// Запуск HTTP-сервера
func startHTTPServer() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}

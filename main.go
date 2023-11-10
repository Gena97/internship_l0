package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v4"
	"github.com/nats-io/stan.go"
)

var db *pgx.Conn

// Order структура данных для заказа
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

var orderCache = make(map[string]Order)

const (
	natsURL     = "nats://localhost:4222"
	clusterID   = "test-cluster"
	clientID    = "your_client_id_2"
	subject     = "your_subject"
	durableName = "durable-order-sub"
)

func main() {
	connectToDB()
	restoreCacheFromDB()
	go subscribeToNATS()
	go sender() // Запуск функции sender в отдельной горутине
	http.HandleFunc("/order", getOrderHandler)
	go startHTTPServer()

	// Ожидание завершения работы (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func connectToDB() {
	connStr := "user=Admin_Gena password=pass dbname=orders_db sslmode=disable port=5433"
	var err error
	db, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatal(err)
	}
}

func restoreCacheFromDB() {
	rows, err := db.Query(context.Background(), "SELECT order_uid, track_number FROM orders")
	if err != nil {
		log.Printf("Error querying orders from PostgreSQL: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.OrderUID, &order.TrackNumber); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		orderCache[order.OrderUID] = order
	}
}

func subscribeToNATS() {
	sc, err := stan.Connect(clusterID, clientID)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	subscription, err := sc.Subscribe(subject, func(msg *stan.Msg) {
		var orderData Order
		if err := json.Unmarshal(msg.Data, &orderData); err != nil {
			log.Printf("Error unmarshalling order data: %v", err)
			return
		}

		// Проверка валидности JSON
		if !json.Valid(msg.Data) {
			log.Printf("Received invalid JSON: %s", msg.Data)
			return
		}

		// Сохранение данных в базе данных и кэше
		saveOrder(orderData)
	}, stan.DurableName(durableName))

	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Unsubscribe()

	// Бесконечный цикл для поддержания подписки
	select {}
}

func saveOrder(order Order) {
	// Сохранение в базе данных
	_, err := db.Exec(context.Background(), "INSERT INTO orders (order_uid, track_number) VALUES ($1, $2)", order.OrderUID, order.TrackNumber)
	if err != nil {
		log.Printf("Error saving order to PostgreSQL: %v", err)
	}

	// Сохранение в кэше
	orderCache[order.OrderUID] = order
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("id")
	order, ok := orderCache[orderID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Order not found"))
		return
	}

	// Отправить данные заказа в ответ на HTTP-запрос
	json.NewEncoder(w).Encode(order)
}
func startHTTPServer() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}

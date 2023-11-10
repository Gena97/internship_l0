package main

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/nats-io/stan.go"
)

func sender() {
	sc, err := stan.Connect(clusterID, "cliend_id_1")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			orderData := Order{
				OrderUID:    strconv.FormatInt(time.Now().UnixNano(), 10), // Генерация уникального идентификатора заказа
				TrackNumber: "2",
				Entry:       "3",
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
					Phone:   "123-456-7890",
					Zip:     "12345",
					City:    "Example City",
					Address: "123 Main St",
					Region:  "Example Region",
					Email:   "john@example.com",
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
					Transaction:  "your_transaction",
					RequestID:    "your_request_id",
					Currency:     "USD",
					Provider:     "Example Provider",
					Amount:       123.45,
					PaymentDT:    time.Now().Unix(),
					Bank:         "Example Bank",
					DeliveryCost: 10.0,
					GoodsTotal:   100.0,
					CustomFee:    5.0,
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
						TrackNumber: "item_track_number",
						Price:       50.0,
						RID:         "item_rid",
						Name:        "Item 1",
						Sale:        0,
						Size:        "M",
						TotalPrice:  50.0,
						NmID:        101,
						Brand:       "Example Brand",
						Status:      1,
					},
					{
						ChrtID:      2,
						TrackNumber: "item_track_number_2",
						Price:       70.0,
						RID:         "item_rid_2",
						Name:        "Item 2",
						Sale:        10,
						Size:        "L",
						TotalPrice:  63.0,
						NmID:        102,
						Brand:       "Example Brand 2",
						Status:      2,
					},
				},
				Locale:            "en",
				InternalSignature: "your_internal_signature",
				CustomerID:        "your_customer_id",
				DeliveryService:   "Example Delivery Service",
				ShardKey:          "your_shardkey",
				SmID:              1,
				DateCreated:       time.Now().Format(time.RFC3339),
				OOFShard:          "your_oof_shard",
			}

			orderJSON, err := json.Marshal(orderData)
			if err != nil {
				log.Println("Error marshalling order data:", err)
				continue
			}

			err = sc.Publish(subject, orderJSON)
			if err != nil {
				log.Println("Error publishing order:", err)
				continue
			}

			log.Println("Order published successfully. OrderUID:", orderData.OrderUID)
		}
	}
}

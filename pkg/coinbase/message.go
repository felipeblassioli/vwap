package coinbase

type MessageChannel struct {
	Name       string   `json:"name"`
	ProductIds []string `json:"product_ids"`
}

// Subscribe message are used to begin receiving feed messages.
// It indicates  which channels and products to receive.
//
// If this message is not sent within 5 seconds, the websocket connection is
// closed by the server.
type Subscribe struct {
	Type     string           `json:"type"`
	Channels []MessageChannel `json:"channels"`
}

// Subscriptions message is the server response to a Subscribe message.
// The subscriptions message lists all channels you are subscribed to.
// Subsequent subscribe messages add to the list of subscriptions.
//
// Example message:
//
//	{
//	   "type": "subscriptions",
//	   "channels": [
//	       {
//	           "name": "level2",
//	           "product_ids": [
//	               "ETH-USD",
//	               "ETH-EUR"
//	           ],
//	       },
//	       {
//	           "name": "heartbeat",
//	           "product_ids": [
//	               "ETH-USD",
//	               "ETH-EUR"
//	           ],
//	       },
//	       {
//	           "name": "ticker",
//	           "product_ids": [
//	               "ETH-USD",
//	               "ETH-EUR",
//	               "ETH-BTC"
//	           ]
//	       }
//	   ]
//	}
type Subscriptions struct {
	// Type is always "subscriptions"
	Type     string           `json:"type"`
	Channels []MessageChannel `json:"channels"`
}

// Unsubscribe message is used to unsubscribe from one or more products from a
// channel.
// The structure is equivalent to Subscribe messages.
//
// An empty product IDs list unsubscribes from a channel entirely.
// Example request:
//
//	{
//	   "type": "unsubscribe",
//	   "channels": [
//	       "matches"
//	   ]
//	}
type Unsubscribe struct {
	// Type is always "unsubscribe"
	Type       string   `json:"type"`
	ProductIds []string `json:"product_ids"`
	Channels   []string `json:"channels"`
}

type Message struct {
	Type         string           `json:"type"`
	ProductID    string           `json:"product_id"`
	ProductIds   []string         `json:"product_ids"`
	TradeID      int              `json:"trade_id"`
	OrderID      string           `json:"order_id"`
	Sequence     int64            `json:"sequence"`
	MakerOrderID string           `json:"maker_order_id"`
	TakerOrderID string           `json:"taker_order_id"`
	Size         string           `json:"size"`
	Price        string           `json:"price"`
	Side         string           `json:"side"`
	Reason       string           `json:"reason"`
	Message      string           `json:"message"`
	Channels     []MessageChannel `json:"channels"`
	LastTradeID  int              `json:"last_trade_id"`
}

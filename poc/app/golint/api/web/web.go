// Package web is the HTTP API.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"stockroom/lib/shop"
)

// Settings say where the API listens.
type Settings struct {
	Addr string
}

// API serves the HTTP API.
type API struct {
	catalog    Catalog
	orders     Orders
	purchasing Purchasing
	events     Events
}

// New returns the API.
func New(catalog Catalog, orders Orders, purchasing Purchasing, events Events) *API {
	return &API{catalog: catalog, orders: orders, purchasing: purchasing, events: events}
}

// Serve serves the API until ctx is cancelled, then lets the requests in progress finish.
func (a *API) Serve(ctx context.Context, s Settings) error {
	srv := &http.Server{Addr: s.Addr, Handler: a.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	log.Printf("stockroom listens on %s", ln.Addr())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}

// Handler returns the routes of the API.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /products", a.addProduct)
	mux.HandleFunc("GET /products", a.listProducts)
	mux.HandleFunc("GET /products/{sku}", a.getProduct)
	mux.HandleFunc("POST /products/{sku}/receive", a.receive)
	mux.HandleFunc("POST /orders", a.createOrder)
	mux.HandleFunc("GET /orders", a.listOrders)
	mux.HandleFunc("GET /orders/{id}", a.getOrder)
	mux.HandleFunc("POST /orders/{id}/pay", orderAction(a.orders.Pay))
	mux.HandleFunc("POST /orders/{id}/ship", orderAction(a.orders.Ship))
	mux.HandleFunc("POST /orders/{id}/cancel", orderAction(a.orders.Cancel))
	mux.HandleFunc("POST /orders/{id}/refunds", a.refund)
	mux.HandleFunc("GET /orders/{id}/payments", a.payments)
	mux.HandleFunc("POST /suppliers", a.addSupplier)
	mux.HandleFunc("GET /suppliers", a.listSuppliers)
	mux.HandleFunc("GET /suppliers/{id}", a.getSupplier)
	mux.HandleFunc("GET /suppliers/{id}/purchase-orders", a.listPurchases)
	mux.HandleFunc("POST /purchase-orders", a.createPurchase)
	mux.HandleFunc("GET /purchase-orders/{id}", a.getPurchase)
	mux.HandleFunc("POST /purchase-orders/{id}/receive", a.receivePurchase)
	mux.HandleFunc("POST /purchase-orders/{id}/cancel", a.cancelPurchase)
	mux.HandleFunc("POST /webhooks", a.subscribe)
	mux.HandleFunc("GET /webhooks", a.listSubscriptions)
	mux.HandleFunc("GET /webhooks/{id}", a.getSubscription)
	mux.HandleFunc("DELETE /webhooks/{id}", a.unsubscribe)
	mux.HandleFunc("GET /webhooks/{id}/deliveries", a.deliveries)
	mux.HandleFunc("POST /webhooks/{id}/retry", a.retry)
	return mux
}

// The JSON forms of the API (docs/poc/stockroom.md).

type productJSON struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Price     int64  `json:"price_cents"`
	OnHand    int    `json:"on_hand"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
}

type lineJSON struct {
	SKU   string `json:"sku"`
	Qty   int    `json:"qty"`
	Price int64  `json:"price_cents"`
}

type orderJSON struct {
	ID        string     `json:"id"`
	Customer  string     `json:"customer"`
	Status    string     `json:"status"`
	Lines     []lineJSON `json:"lines"`
	Total     int64      `json:"total_cents"`
	Charged   int64      `json:"charged_cents"`
	Refunded  int64      `json:"refunded_cents"`
	PaymentID string     `json:"payment_id"`
}

type paymentJSON struct {
	Kind       string `json:"kind"`
	Amount     int64  `json:"amount_cents"`
	ProviderID string `json:"provider_id"`
}

type supplierJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type purchaseLineJSON struct {
	SKU      string `json:"sku"`
	Qty      int    `json:"qty"`
	Received int    `json:"received"`
	Cost     int64  `json:"cost_cents"`
}

type purchaseJSON struct {
	ID       string             `json:"id"`
	Supplier string             `json:"supplier_id"`
	Status   string             `json:"status"`
	Lines    []purchaseLineJSON `json:"lines"`
	Total    int64              `json:"total_cents"`
}

type subscriptionJSON struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret,omitempty"`
}

type deliveryJSON struct {
	EventID  string `json:"event_id"`
	Type     string `json:"type"`
	Subject  string `json:"subject"`
	Status   string `json:"status"`
	Attempts int    `json:"attempts"`
	Response int    `json:"response_status"`
}

func itemJSON(it shop.Item) productJSON {
	return productJSON{
		SKU: string(it.Product.SKU), Name: it.Product.Name, Price: int64(it.Product.Price),
		OnHand: it.Stock.OnHand, Reserved: it.Stock.Reserved, Available: it.Stock.Available(),
	}
}

func toOrderJSON(o shop.Order) orderJSON {
	out := orderJSON{
		ID: string(o.ID), Customer: string(o.Customer), Status: string(o.Status), Lines: []lineJSON{},
		Total: int64(o.Total), Charged: int64(o.Charged), Refunded: int64(o.Refunded), PaymentID: o.PaymentID,
	}
	for _, l := range o.Lines {
		out.Lines = append(out.Lines, lineJSON{SKU: string(l.SKU), Qty: l.Qty, Price: int64(l.Price)})
	}
	return out
}

func toPurchaseJSON(po shop.PurchaseOrder) purchaseJSON {
	out := purchaseJSON{ID: string(po.ID), Supplier: string(po.Supplier), Status: string(po.Status), Lines: []purchaseLineJSON{}, Total: int64(po.Total)}
	for _, l := range po.Lines {
		out.Lines = append(out.Lines, purchaseLineJSON{SKU: string(l.SKU), Qty: l.Qty, Received: l.Received, Cost: int64(l.Cost)})
	}
	return out
}

// toSubscriptionJSON leaves out the secret: only the response that creates a subscription
// shows it.
func toSubscriptionJSON(sub shop.Subscription) subscriptionJSON {
	out := subscriptionJSON{ID: string(sub.ID), URL: sub.URL, Events: []string{}}
	for _, t := range sub.Events {
		out.Events = append(out.Events, string(t))
	}
	return out
}

func toDeliveriesJSON(list []shop.Delivery) []deliveryJSON {
	out := []deliveryJSON{}
	for _, d := range list {
		out = append(out, deliveryJSON{
			EventID: string(d.Event.ID), Type: string(d.Event.Type), Subject: d.Event.Subject,
			Status: string(d.Status), Attempts: d.Attempts, Response: d.Response,
		})
	}
	return out
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// fail writes an error response: the code of a shop error gives its status.
func fail(w http.ResponseWriter, err error) {
	code := shop.CodeOf(err)
	status := map[string]int{
		shop.CodeInvalid:     http.StatusBadRequest,
		shop.CodeNotFound:    http.StatusNotFound,
		shop.CodeConflict:    http.StatusConflict,
		shop.CodeOutOfStock:  http.StatusConflict,
		shop.CodeDeclined:    http.StatusPaymentRequired,
		shop.CodeUnavailable: http.StatusBadGateway,
	}[code]
	if status == 0 {
		log.Printf("internal error: %v", err)
		code, status = "internal", http.StatusInternalServerError
	}
	msg := err.Error()
	var e *shop.Error
	if errors.As(err, &e) {
		msg = e.Message
	}
	reply(w, status, map[string]string{"error": code, "message": msg})
}

// decode reads a JSON body; an unknown field or a bad value is invalid.
func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return shop.Errorf(shop.CodeInvalid, "bad request body: %v", err)
	}
	return nil
}

func (a *API) addProduct(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SKU   string `json:"sku"`
		Name  string `json:"name"`
		Price int64  `json:"price_cents"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	it, err := a.catalog.AddProduct(shop.Product{SKU: shop.SKU(in.SKU), Name: in.Name, Price: shop.Money(in.Price)})
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, itemJSON(it))
}

func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	items, err := a.catalog.Products()
	if err != nil {
		fail(w, err)
		return
	}
	out := []productJSON{}
	for _, it := range items {
		out = append(out, itemJSON(it))
	}
	reply(w, http.StatusOK, out)
}

func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	it, err := a.catalog.Product(shop.SKU(r.PathValue("sku")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, itemJSON(it))
}

func (a *API) receive(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Qty int `json:"qty"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	it, err := a.catalog.Receive(shop.SKU(r.PathValue("sku")), in.Qty)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, itemJSON(it))
}

func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Customer string `json:"customer"`
		Lines    []struct {
			SKU string `json:"sku"`
			Qty int    `json:"qty"`
		} `json:"lines"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	var wants []shop.Want
	for _, l := range in.Lines {
		wants = append(wants, shop.Want{SKU: shop.SKU(l.SKU), Qty: l.Qty})
	}
	o, err := a.orders.Create(shop.CustomerID(in.Customer), wants)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, toOrderJSON(o))
}

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	customer := strings.TrimSpace(r.URL.Query().Get("customer"))
	if customer == "" {
		fail(w, shop.Errorf(shop.CodeInvalid, "the query needs customer"))
		return
	}
	list, err := a.orders.List(shop.CustomerID(customer))
	if err != nil {
		fail(w, err)
		return
	}
	out := []orderJSON{}
	for _, o := range list {
		out = append(out, toOrderJSON(o))
	}
	reply(w, http.StatusOK, out)
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.orders.Get(shop.OrderID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toOrderJSON(o))
}

func orderAction(act func(shop.OrderID) (shop.Order, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		o, err := act(shop.OrderID(r.PathValue("id")))
		if err != nil {
			fail(w, err)
			return
		}
		reply(w, http.StatusOK, toOrderJSON(o))
	}
}

func (a *API) refund(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount int64 `json:"amount_cents"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	o, err := a.orders.Refund(shop.OrderID(r.PathValue("id")), shop.Money(in.Amount))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toOrderJSON(o))
}

func (a *API) payments(w http.ResponseWriter, r *http.Request) {
	list, err := a.orders.Payments(shop.OrderID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	out := []paymentJSON{}
	for _, p := range list {
		out = append(out, paymentJSON{Kind: p.Kind, Amount: int64(p.Amount), ProviderID: p.ProviderID})
	}
	reply(w, http.StatusOK, out)
}

func (a *API) addSupplier(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	sup, err := a.purchasing.AddSupplier(in.Name)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, supplierJSON{ID: string(sup.ID), Name: sup.Name})
}

func (a *API) listSuppliers(w http.ResponseWriter, r *http.Request) {
	list, err := a.purchasing.Suppliers()
	if err != nil {
		fail(w, err)
		return
	}
	out := []supplierJSON{}
	for _, sup := range list {
		out = append(out, supplierJSON{ID: string(sup.ID), Name: sup.Name})
	}
	reply(w, http.StatusOK, out)
}

func (a *API) getSupplier(w http.ResponseWriter, r *http.Request) {
	sup, err := a.purchasing.Supplier(shop.SupplierID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, supplierJSON{ID: string(sup.ID), Name: sup.Name})
}

func (a *API) listPurchases(w http.ResponseWriter, r *http.Request) {
	list, err := a.purchasing.List(shop.SupplierID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	out := []purchaseJSON{}
	for _, po := range list {
		out = append(out, toPurchaseJSON(po))
	}
	reply(w, http.StatusOK, out)
}

func (a *API) createPurchase(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Supplier string `json:"supplier_id"`
		Lines    []struct {
			SKU  string `json:"sku"`
			Qty  int    `json:"qty"`
			Cost int64  `json:"cost_cents"`
		} `json:"lines"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	var wants []shop.PurchaseWant
	for _, l := range in.Lines {
		wants = append(wants, shop.PurchaseWant{SKU: shop.SKU(l.SKU), Qty: l.Qty, Cost: shop.Money(l.Cost)})
	}
	po, err := a.purchasing.Create(shop.SupplierID(in.Supplier), wants)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, toPurchaseJSON(po))
}

func (a *API) getPurchase(w http.ResponseWriter, r *http.Request) {
	po, err := a.purchasing.Get(shop.PurchaseID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toPurchaseJSON(po))
}

func (a *API) receivePurchase(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Lines []struct {
			SKU string `json:"sku"`
			Qty int    `json:"qty"`
		} `json:"lines"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	var arrivals []shop.Arrival
	for _, l := range in.Lines {
		arrivals = append(arrivals, shop.Arrival{SKU: shop.SKU(l.SKU), Qty: l.Qty})
	}
	po, err := a.purchasing.Receive(shop.PurchaseID(r.PathValue("id")), arrivals)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toPurchaseJSON(po))
}

func (a *API) cancelPurchase(w http.ResponseWriter, r *http.Request) {
	po, err := a.purchasing.Cancel(shop.PurchaseID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toPurchaseJSON(po))
}

func (a *API) subscribe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, err)
		return
	}
	var types []shop.EventType
	for _, t := range in.Events {
		types = append(types, shop.EventType(t))
	}
	sub, err := a.events.Subscribe(in.URL, types)
	if err != nil {
		fail(w, err)
		return
	}
	out := toSubscriptionJSON(sub)
	out.Secret = sub.Secret
	reply(w, http.StatusCreated, out)
}

func (a *API) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	list, err := a.events.Subscriptions()
	if err != nil {
		fail(w, err)
		return
	}
	out := []subscriptionJSON{}
	for _, sub := range list {
		out = append(out, toSubscriptionJSON(sub))
	}
	reply(w, http.StatusOK, out)
}

func (a *API) getSubscription(w http.ResponseWriter, r *http.Request) {
	sub, err := a.events.Subscription(shop.SubscriptionID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toSubscriptionJSON(sub))
}

func (a *API) unsubscribe(w http.ResponseWriter, r *http.Request) {
	if err := a.events.Unsubscribe(shop.SubscriptionID(r.PathValue("id"))); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) deliveries(w http.ResponseWriter, r *http.Request) {
	list, err := a.events.Deliveries(shop.SubscriptionID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toDeliveriesJSON(list))
}

func (a *API) retry(w http.ResponseWriter, r *http.Request) {
	list, err := a.events.Retry(shop.SubscriptionID(r.PathValue("id")))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, toDeliveriesJSON(list))
}

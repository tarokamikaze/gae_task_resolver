package main

import (
	"net/http"
	"fmt"
	"google.golang.org/appengine"
	"sync"
	"context"
	"cloud.google.com/go/datastore"
	"os"
	"google.golang.org/api/iterator"
)

const (
	stNone  = iota
	stDoing
	stDone
)

type (
	Item struct {
		ID    string
		State int
		Score int
	}
)

var mu = new(sync.Mutex)

func main() {

	http.HandleFunc("/get", GetHandler)
	http.HandleFunc("/finished", FinishedHandler)

	// health check
	healthCheckHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}
	http.HandleFunc("/liveness_check", healthCheckHandler)
	http.HandleFunc("/readiness_check", healthCheckHandler)
	appengine.Main()
}
func GetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	cli, err := datastore.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	item, err := getItem(ctx, cli)
	if err != nil {
		if err == iterator.Done {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}
	item.State = stDoing
	key := datastore.NameKey("Item", item.ID, nil)

	if _, err := cli.Put(ctx, key, item); err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, item.ID)
}
func FinishedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	vals, ok := r.URL.Query()["id"]
	if !ok {
		http.Error(w, fmt.Sprintf("id is required"), http.StatusInternalServerError)
		return
	}
	id := vals[0]
	if id == "" {
		http.Error(w, fmt.Sprintf("id is required"), http.StatusInternalServerError)
		return
	}

	cli, err := datastore.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}

	key := datastore.NameKey("Item", id, nil)
	item := &Item{}
	if err := cli.Get(ctx, key, item); err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}
	item.State = stDone
	if _, err := cli.Put(ctx, key, item); err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, item.ID)
}
func getItem(ctx context.Context, cli *datastore.Client) (*Item, error) {

	q := datastore.NewQuery("Item").
		Filter("State = ", stNone).
		Order("Score").
		Limit(1)

	it := cli.Run(ctx, q)
	for {
		res := &Item{}
		_, err := it.Next(res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
}

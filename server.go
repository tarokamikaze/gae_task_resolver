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
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"encoding/json"
	"math/rand"
	"time"
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

var getMu = new(sync.Mutex)
var cli = &datastore.Client{}

func main() {

	ctx := context.Background()
	cl, err := datastore.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		println(err.Error())
		return
	}
	cli = cl
	defer cli.Close()

	http.HandleFunc("/get", GetHandler)
	http.HandleFunc("/add", AddHandler)
	http.HandleFunc("/finished", FinishedHandler)
	http.HandleFunc("/state", StateHandler)

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

	getMu.Lock()
	defer getMu.Unlock()
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
func StateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	states := map[int]string{
		stNone:  "none",
		stDoing: "doing",
		stDone:  "done",
	}
	counts := map[int]int{}
	mu := new(sync.Mutex)
	eg := errgroup.Group{}
	egHandler := func(state int, str string) func() error {
		return func() error {
			q := datastore.NewQuery("Item").
				Filter("State = ", state).
				KeysOnly()

			cnt, err := cli.Count(ctx, q)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			counts[state] = cnt
			return nil
		}
	}
	for s, v := range states {
		eg.Go(egHandler(s, v))
	}
	err := eg.Wait()
	if err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}

	res := ""
	for i := 0; i <= stDone; i++ {
		v := states[i]
		cnt := counts[i]
		res += fmt.Sprintf(" %s:%d", v, cnt)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, res)
}
func AddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	item := &Item{}
	err := json.Unmarshal(body, item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if item.ID == "" {
		http.Error(w, fmt.Sprintf("id is required"), http.StatusInternalServerError)
		return
	}

	key := datastore.NameKey("Item", item.ID, nil)
	if err := cli.Get(ctx, key, item); err == nil {
		http.Error(w, fmt.Sprintf("the item has already exists: %s", item.ID), http.StatusInternalServerError)
		return
	}

	rand.Seed(time.Now().UnixNano())
	item.State = stNone
	item.Score = rand.Intn(1000000)
	if _, err := cli.Put(ctx, key, item); err != nil {
		http.Error(w, fmt.Sprintf("An error was occured: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, item.ID)
}
func FinishedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	body, _ := ioutil.ReadAll(r.Body)
	item := &Item{}
	err := json.Unmarshal(body, item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if item.ID == "" {
		http.Error(w, fmt.Sprintf("id is required"), http.StatusInternalServerError)
		return
	}

	key := datastore.NameKey("Item", item.ID, nil)
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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
)

type handler struct {
	resource interface{}
}

func (handler *handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(resp, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	jsonBytes, err := json.MarshalIndent(handler.resource, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(resp, string(jsonBytes)+"\n")
}

type server struct {
	*http.Server
}

func NewServer(port string) *server {
	addr := "0.0.0.0:" + port
	httpServer := http.Server{Addr: addr, Handler: http.DefaultServeMux}
	return &server{&httpServer}
}

func (server *server) Bind(resource map[string]interface{}) {
	for key, value := range resource {
		http.Handle("/"+key, &handler{resource: value})
	}
}

func (server *server) Start() {
	log.Printf("Listening on %s\n", server.Addr)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Panic(err)
		}
	}()
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	sig := <-quit
	log.Println("Shutting down server... Reason:", sig)

	if err := server.Shutdown(context.Background()); err != nil {
		panic(err)
	}
	log.Println("Server gracefully stopped")
}

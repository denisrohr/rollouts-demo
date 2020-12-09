package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// defaultTerminationDelay delays termination of the program in a graceful shutdown situation.
	// We do this to prevent the pod from exiting immediately upon a pod termination event
	// (e.g. during a rolling update). This gives some time for ingress controllers to react to
	// the Pod IP being removed from the Service's Endpoint list, which prevents traffic from being
	// directed to terminated pods, which otherwise would cause timeout errors and/or request delays.
	// See: See: https://github.com/kubernetes/ingress-nginx/issues/3335#issuecomment-434970950
	defaultTerminationDelay = 10
)

var (
	color  = os.Getenv("COLOR")
	colors = []string{
		"red",
		"orange",
		"yellow",
		"green",
		"blue",
		"purple",
	}
)

func main() {
	var (
		listenAddr       string
		terminationDelay int
		numCPUBurn       string
	)
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "server listen address")
	flag.IntVar(&terminationDelay, "termination-delay", defaultTerminationDelay, "termination delay in seconds")
	flag.StringVar(&numCPUBurn, "cpu-burn", "", "burn specified number of cpus (number or 'all')")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	router := http.NewServeMux()
	router.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./"))))
	router.HandleFunc("/color", getColor)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		server.SetKeepAlivesEnabled(false)
		log.Printf("Signal %v caught. Shutting down in %vs", sig, terminationDelay)
		delay := time.NewTicker(time.Duration(terminationDelay) * time.Second)
		defer delay.Stop()
		select {
		case <-quit:
			log.Println("Second signal caught. Shutting down NOW")
		case <-delay.C:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Printf("Started server on %s", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	log.Println("Server stopped")
}

type colorParameters struct {
	Color            string `json:"color"`
	DelayProbability *int   `json:"delayPercent,omitempty"`
	DelayLength      int    `json:"delayLength,omitempty"`

	Return500Probability *int `json:"return500,omitempty"`
}

func getColor(w http.ResponseWriter, r *http.Request) {
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err.Error())
		fmt.Fprintf(w, err.Error())
		return
	}

	colorToReturn := randomColor()
	if color != "" {
		colorToReturn = color
	}

	printColor(colorToReturn, w, true)
}

func printColor(colorToPrint string, w http.ResponseWriter, healthy bool) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		log.Println("Returning 500")
		w.WriteHeader(500)
	}
	switch colorToPrint {
	case "":
		randomColor := randomColor()
		if healthy {
			log.Printf("Successful %s\n", randomColor)
		} else {
			log.Printf("500 - %s\n", randomColor)
		}
		fmt.Fprintf(w, "\"%s\"", randomColor)
	default:
		if healthy {
			log.Printf("Successful %s\n", colorToPrint)
		} else {
			log.Printf("500 - %s\n", colorToPrint)
		}
		fmt.Fprintf(w, "\"%s\"", colorToPrint)
	}
}

func randomColor() string {
	return colors[rand.Int()%len(colors)]
}

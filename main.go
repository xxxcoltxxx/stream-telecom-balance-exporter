package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

var addr = flag.String("listen-address", "0.0.0.0:9602", "The address to listen on for HTTP requests.")
var interval = flag.Int("interval", 3600, "Interval (in seconds) for request balance.")
var retryInterval = flag.Int("retry-interval", 10, "Interval (in seconds) for load balance when errors.")
var retryLimit = flag.Int("retry-limit", 10, "Count of tries when error.")

var (
	credentials  = CredentialsConfig{}
	balanceGauge *prometheus.GaugeVec
	hasError     = false
	retryCount   = 0
)

// CredentialsConfig is credentials for access to an API
type CredentialsConfig struct {
	Login    string
	Password string
}

func init() {
	balanceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "balance",
			Name:      "stream_telecom",
			Help:      "Balance in Stream Telecom account",
		},
		[]string{"service"},
	)

	prometheus.MustRegister(balanceGauge)

	flag.Parse()
}

func main() {
	log.Println("Starting Stream Telecom balance exporter", version.Info())
	log.Println("Build context", version.BuildContext())

	if err := readConfig(); err != nil {
		log.Fatalln("Configuration error:", err.Error())
	}

	if err := loadBalance(); err != nil {
		log.Fatalln(err.Error())
	}

	go startBalanceUpdater()

	srv := &http.Server{
		Addr:         *addr,
		WriteTimeout: time.Second * 2,
		ReadTimeout:  time.Second * 2,
		IdleTimeout:  time.Second * 60,

		Handler: nil,
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	log.Printf("Stream Telecom balance exporter has been started at address %s\n", *addr)
	log.Printf("Exporter will update balance every %d seconds\n", *interval)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	<-c

	log.Println("Stream Telecom balance exporter shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

func readConfig() error {
	if login, ok := os.LookupEnv("STREAM_TELECOM_LOGIN"); ok {
		credentials.Login = login
	} else {
		return errors.New("environment \"STREAM_TELECOM_LOGIN\" is not set")
	}

	if password, ok := os.LookupEnv("STREAM_TELECOM_PASSWORD"); ok {
		credentials.Password = password
	} else {
		return errors.New("environment \"STREAM_TELECOM_PASSWORD\" is not set")
	}

	return nil
}

func startBalanceUpdater() {
	for {
		if hasError {
			log.Printf("Request will retry after %d seconds\n", *retryInterval)
			time.Sleep(time.Second * time.Duration(*retryInterval))
		} else {
			time.Sleep(time.Second * time.Duration(*interval))
		}

		if err := loadBalance(); err != nil {
			log.Println(err.Error())
			hasError = true
			retryCount++
			if retryCount >= *retryLimit {
				log.Printf("Retry limit %d has been exceeded\n", *retryLimit)
				hasError = false
				retryCount = 0
			}
		} else {
			hasError = false
			retryCount = 0
		}
	}
}

func hideCredentials(format string, args ...interface{}) string {
	var message = fmt.Sprintf(format, args...)
	message = strings.Replace(message, credentials.Login, "<stream-telecom-login>", -1)
	message = strings.Replace(message, credentials.Password, "<stream-telecom-password>", -1)

	return message
}

func loadBalance() error {
	body, err := loadBody()
	if err != nil {
		return err
	}

	if b, err := strconv.ParseFloat(body, 2); err != nil {
		return errors.New(hideCredentials("Cannot parse balance: %s", err.Error()))
	} else {
		balanceGauge.With(prometheus.Labels{"service": credentials.Login}).Set(b)
	}

	return nil
}

func loadBody() (string, error) {
	client := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, "https://gateway.api.sc/get/", nil)
	q := req.URL.Query()
	q.Add("user", credentials.Login)
	q.Add("pwd", credentials.Password)
	q.Add("balance", "1")
	req.URL.RawQuery = q.Encode()

	if err != nil {
		return "", errors.New(hideCredentials("Cannot create request: %s", err.Error()))
	}

	res, err := client.Do(req)
	if err != nil {
		return "", errors.New(hideCredentials("Request error: %s", err.Error()))
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Println(hideCredentials("Cannot close response body: %s", err.Error()))
		}
	}()

	scanner := bufio.NewScanner(res.Body)
	ok := scanner.Scan()
	body := scanner.Text()
	if !ok {
		return "", errors.New(hideCredentials("Cannot parse response body: %s", body))
	}

	return body, nil
}

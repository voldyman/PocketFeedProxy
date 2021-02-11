package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

const mimeRSSXML = "application/rss+xml"

var log *zap.Logger

var (
	listenFlag = flag.String("listen", "localhost:9092", "address to listen on")
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := installLogging(); err != nil {
		return err
	}

	config, err := readConfig()
	if err != nil {
		return err
	}

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "you shouldn't be here")
	})

	r.GET("/feed", func(c *gin.Context) {
		feed, err := fetchPocketFeed(config)
		if err != nil {
			log.Error("fetching feed failed", zap.Error(err))
			c.String(http.StatusInternalServerError, "unable to fetch feed")
		}
		c.Data(http.StatusOK, mimeRSSXML, feed)
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	return r.Run(config.listenAddr)
}

func fetchPocketFeed(cfg *config) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", cfg.pocketURL, nil)
	req.SetBasicAuth(cfg.username, cfg.password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to execute request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to ready body")
	}

	return body, nil
}
func installLogging() error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	log = logger
	return nil
}

type config struct {
	listenAddr string

	pocketURL string
	username  string
	password  string
}

func readConfig() (*config, error) {
	flag.Parse()
	username, password, err := readCredentials()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read credentials")
	}
	url := fmt.Sprintf("https://getpocket.com/users/%s/feed/unread", username)
	return &config{
		pocketURL:  url,
		username:   username,
		password:   password,
		listenAddr: *listenFlag,
	}, nil
}
func readCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", err
	}

	fmt.Println()
	password := string(bytePassword)
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

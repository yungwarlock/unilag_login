package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"net/http"
	"net/url"

	"gopkg.in/yaml.v3"
	"mvdan.cc/xurls/v2"

	"golang.org/x/net/html"
)

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

const hostURL string = "http://192.0.0.1"
const filePath string = "/etc/unilag_login.yaml"

type LoginData struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func extractLoginUrl() (string, error) {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	res, err := client.Get(hostURL)
	if err != nil {
		return "", errors.New("Already logged in")
	}
	defer res.Body.Close()

	doc, err := html.Parse(res.Body)
	if err != nil {
		return "", err
	}

	var url string
	var crawler func(*html.Node)
	// Crawl the html document to find the login url
	crawler = func(node *html.Node) {
		rxRelaxed := xurls.Relaxed()
		// Extract the login url from the redirect link added in the meta tag
		if node.Type == html.ElementNode && node.Data == "meta" && node.Attr[0].Val == "refresh" {
			url = rxRelaxed.FindString(node.Attr[1].Val)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}

	crawler(doc)
	if url != "" {
		return url, nil
	}

	return "", errors.New("Already logged in")
}

func login(hostURL string, username string, password string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	values := url.Values{}
	values.Add("username", username)
	values.Add("password", password)

	res, err := client.PostForm(hostURL, values)
	if err != nil {
		fmt.Println(err)
		return errors.New("Unable to login")
	} else {
		defer res.Body.Close()
		return nil
	}
}

func getLoginData() (string, string, error) {
	f, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", errors.New("Unable to get login data")
	}

	var loginData LoginData

	if err := yaml.Unmarshal(f, &loginData); err != nil {
		return "", "", err
	}

	return loginData.Username, loginData.Password, nil

}

func main() {
	// Get the login url
	loginURL, err := extractLoginUrl()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the login credentials from storage
	username, password, err := getLoginData()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send the login request
	if err := login(loginURL, username, password); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Login successful")
	}
}

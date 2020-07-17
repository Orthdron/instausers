package main

import (
	"bufio"
	"container/ring"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"
)
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func AppendStringToFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text+"\n")
	if err != nil {
		return err
	}
	return nil
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config.json file: %s \n", err))
	}

	file, err := os.Open(viper.GetString("list"))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	usernamechannel := make(chan string)
	proxychannel := make(chan string)
	removechannel := make(chan string)
	go func() {
		for{
			removethis := <- removechannel
			userlist := viper.GetString("list")
			out , err := exec.Command("sed","-i'.bak'","-e","s/"+ removethis + "//g",userlist).CombinedOutput()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + string(out))
			} else{
				output := string(out[:])
				fmt.Println(output)
			}
		}
	}()
	for i := 0; i <= viper.GetInt("threads") ; i++  {
		go func(){
			for {
				usernameprovided := <-usernamechannel
				urlStr := "https://www.instagram.com/web/search/topsearch/?query=" + usernameprovided
				url, err := url.Parse(urlStr)
				if err != nil {
					log.Println(err)
				}
				proxyStr := "http://" + <-proxychannel
				proxyURL, err := url.Parse(proxyStr)
				if err != nil {
					log.Println(err)
				}
				transport := &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				}
				client := &http.Client{
					Transport: transport,
				}
				request, err := http.NewRequest("GET", url.String(), nil)
				if err != nil {
					log.Println(err)
				}
				response, err := client.Do(request)
				if err != nil {
					log.Println(err)
				} else{
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						log.Println(err)
					}
					bytedata := []byte(data)
					username,_ := jsonparser.GetString(bytedata, "users", "[0]", "user", "username")
					if usernameprovided == username{
						fmt.Print(username + " - ")
						followers , _ := jsonparser.GetInt(bytedata, "users", "[0]", "user", "follower_count")
						if err != nil{
							log.Println(err)
						}
						fmt.Println(followers)
						if followers > 0{
							removechannel <- username
						}
						if followers >= 1000{
							AppendStringToFile(viper.GetString("over10kfile"), username)
						}else{
							AppendStringToFile(viper.GetString("under10kfile"), username)
						}
					}
					defer response.Body.Close()
				}
			}
		}()
	}
	proxies, err := readLines(viper.GetString("proxyfile"))
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}
	proxyring := ring.New(len(proxies))
	for i := 0; i < proxyring.Len(); i++ {
		proxyring.Value = proxies[i]
		proxyring = proxyring.Next()
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() != "" {
			usernamechannel <- scanner.Text()
			proxychannel <- proxyring.Value.(string)
			delay := viper.GetInt("delay")
			time.Sleep(time.Duration(delay) * time.Second)
			proxyring = proxyring.Next()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
package main

import (
	"os"
	"io"
	"fmt"
	"log"
	"strings"
	"sync"
	"io/ioutil"
	"encoding/json"
	"encoding/csv"
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
	"net/http"
	"errors"
)

const BASE_URL string = "https://m.avito.ru"
var IPBanned error = errors.New("You are banned by Avito!")


type Item struct {
	header   string
	location string
	url      string
	phone    string
}
var (
	Info    *log.Logger
	Error   *log.Logger
)

func perror(err error) {
	if err != nil {
		Error.Println(err)
//		panic(err)
	}
}

func InitLoggers(verbose bool) {
	var infoHandle, errorHandle io.Writer

	if verbose {
		infoHandle = os.Stdout
		errorHandle = os.Stderr
	}else {
		infoHandle = ioutil.Discard
		errorHandle = os.Stderr
	}

	Info = log.New(infoHandle, "INFO: ", 0)
	Error = log.New(errorHandle, "ERROR: ", 0)
}

// Достаёт телефонный номер из JSON по заданному URL
func getPhone(phone_url, referer string) (string, error) {
	Info.Println("Persing phone url:", phone_url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", phone_url, nil)
	req.Header.Add("referer", referer)

	res, err := client.Do(req)
	perror(err)

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	perror(err)
	if res.StatusCode == 403 {
		return "", IPBanned
	}

	phone_data := make(map[string]string)
	err = json.Unmarshal(body, &phone_data)
	perror(err)

	Info.Println("Phone number:", phone_data["phone"])
	return phone_data["phone"], nil
}

func parseItem(item *Item, wg *sync.WaitGroup, items chan *Item) {
	doc, err := goquery.NewDocument(item.url)
	perror(err)

	doc.Find(".action-show-number").Each(func(i int, s *goquery.Selection) {
		phone_url, exists := s.Attr("href")
		if exists {
			Info.Println("Found phone url:", phone_url)

			phone_url := strings.Join([]string{BASE_URL, phone_url, "?async"}, "")
			item.phone, err = getPhone(phone_url, item.url)
			if err != nil {
				Error.Println(err)
			}else {
				items <- item
			}


		}
	})
	wg.Done()
}

func saveToCSV(items chan *Item) {
	w := csv.NewWriter(os.Stdout)

	for item := range items {
		record := []string{item.header, item.location, item.phone, item.url}
		if err := w.Write(record); err != nil {
			Error.Println("error writing record to csv:", err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}



func main() {
	var query string
	var location string
	var category string
	var verbose bool

	var SelaAvitoCmd = &cobra.Command{
		Use: "selavito",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			InitLoggers(verbose)

			items := make(chan *Item)

			go saveToCSV(items)

			var search_url string

//			TODO: добавить паджинацию
			if category == "" {
				search_url = fmt.Sprintf("%s/%s?q=%s", BASE_URL, location, query)
			} else {
				search_url = fmt.Sprintf("%s/%s/%s?q=%s", BASE_URL, location, category, query)
			}

			Info.Print(search_url)

			doc, err := goquery.NewDocument(search_url)
			perror(err)

			wg := new(sync.WaitGroup)

			doc.Find(".b-item").Each(func(i int, s *goquery.Selection) {
				item_url, exists := s.Find(".item-link").Attr("href")
				if exists {
					var item Item
					item.header = s.Find(".header-text").First().Text()
					item.location = s.Find(".info-location").First().Text()
					item.url = fmt.Sprintf("%s%s", BASE_URL, item_url)
					wg.Add(1)
					go parseItem(&item, wg, items)
					Info.Printf("%+v\n", item)
				}

			})

			wg.Wait()
			close(items)
		},
	}

	SelaAvitoCmd.Flags().StringVarP(&query, "query", "q", "", "Query string.")
	SelaAvitoCmd.Flags().StringVarP(&location, "location", "l", "rossiya",
		"Filter by location. Examples: moskva, moskovskaya_oblast, sankt-peterburg.")
	SelaAvitoCmd.Flags().StringVarP(&category, "category", "c", "",
		"Filter by category. Examples: nedvizhimost, transport, rabota, rezume, vakansii.")

	SelaAvitoCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Show more information.")

	SelaAvitoCmd.Execute()

}

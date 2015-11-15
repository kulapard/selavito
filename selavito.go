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
	Debug   *log.Logger
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
	var infoHandle, errorHandle, debugHandle io.Writer

	if verbose {
		debugHandle = os.Stdout
	}else {
		debugHandle = ioutil.Discard
	}

	infoHandle = os.Stdout
	errorHandle = os.Stderr

	Debug = log.New(debugHandle, "DEBUG: ", 0)
	Info = log.New(infoHandle, "INFO: ", 0)
	Error = log.New(errorHandle, "ERROR: ", 0)
}

// Достаёт телефонный номер из JSON по заданному URL
func getPhone(phone_url, referer string) (string, error) {
	Debug.Println("Persing phone url:", phone_url)

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

	Debug.Println("Phone number:", phone_data["phone"])
	return phone_data["phone"], nil
}

func parseItem(item *Item, wg *sync.WaitGroup, items chan *Item) {
	doc, err := goquery.NewDocument(item.url)
	perror(err)

	doc.Find(".action-show-number").Each(func(i int, s *goquery.Selection) {
		phone_url, exists := s.Attr("href")
		if exists {
			Debug.Println("Found phone url:", phone_url)

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

func saveToCSV(items chan *Item, wg *sync.WaitGroup) {
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

	wg.Done()
}



func main() {
	var query string
	var location string
	var category string
	var verbose bool
	var max_items int8

	var SelaAvitoCmd = &cobra.Command{
		Use: "selavito",
		Short: "Утилита для получения телефонных номеров с Avito (avito.ru)",
		Example: "selavito -l moskva -q macbook\nselavito -l sankt-peterburg -с rabota -q golang",

		Run: func(cmd *cobra.Command, args []string) {
			if query == "" {
				cmd.Help()
				return
			}

			var page_url string
			counter := max_items
			items := make(chan *Item)
			save_wg := new(sync.WaitGroup)
			parse_wg := new(sync.WaitGroup)

			InitLoggers(verbose)

			save_wg.Add(1)
			go saveToCSV(items, save_wg)

			if category == "" {
				page_url = fmt.Sprintf("%s/%s?q=%s", BASE_URL, location, query)
			} else {
				page_url = fmt.Sprintf("%s/%s/%s?q=%s", BASE_URL, location, category, query)
			}

			for page_url != "" && counter > 0 {
				Info.Println(page_url)

				doc, err := goquery.NewDocument(page_url)
				perror(err)

				next_page_url, _ := doc.Find(".page-next").Find("a").First().Attr("href")
				next_page_url = fmt.Sprintf("%s%s", BASE_URL, next_page_url)
				Debug.Println("Next page:", next_page_url)

				doc.Find(".b-item").Each(func(i int, s *goquery.Selection) {
					if counter > 0 {
						item_url, exists := s.Find(".item-link").Attr("href")
						if exists {
							var item Item
							item.header = s.Find(".header-text").First().Text()
							item.location = s.Find(".info-location").First().Text()
							item.url = fmt.Sprintf("%s%s", BASE_URL, item_url)
							parse_wg.Add(1)
							go parseItem(&item, parse_wg, items)
							counter--
							Debug.Printf("%+v\n", item)
						}else {
							Error.Println(".item-link not found")
						}
					}
				})
				page_url = next_page_url
			}

			// Дожидаемся завершения работы всех парсеров...
			parse_wg.Wait()

			// ...и только после закрываем канал
			close(items)

			// Ждём пока данные окончательно сохранятся
			save_wg.Wait()
		},
	}

	SelaAvitoCmd.Flags().StringVarP(&query, "query", "q", "", "Строка для поиска")
	SelaAvitoCmd.Flags().StringVarP(&location, "location", "l", "rossiya",
		"Фильтр по региону (примеры: moskva, moskovskaya_oblast, sankt-peterburg)")
	SelaAvitoCmd.Flags().StringVarP(&category, "category", "c", "",
		"Фильтр по категории (примеры: nedvizhimost, transport, rabota, rezume, vakansii)")

	SelaAvitoCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Более подробный вывод в консоль")

	SelaAvitoCmd.Flags().Int8VarP(&max_items, "max", "m", 1,
		"Максимальное количество элементов для поиска")

	SelaAvitoCmd.Execute()

}

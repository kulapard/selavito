package main

import (
	"os"
	"io"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
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
var throttle <-chan time.Time

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

	throttleWait()
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
	throttleWait()

	doc, err := goquery.NewDocument(item.url)
	if err != nil {
		Error.Println(err)
		wg.Done()
		return
	}

	item.location = doc.Find(".avito-address-text").First().Text()
	item.location = strings.TrimSpace(item.location)

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

func saveToCSV(path_to_csvfile string, items chan *Item, wg *sync.WaitGroup) {
	csvfile, err := os.Create(path_to_csvfile)

	if err != nil {
		Error.Println(err)
		return
	}
	defer csvfile.Close()

	w := csv.NewWriter(csvfile)

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


func throttleSet(pause int64) {
	Debug.Printf("Set throttle pause: %d ms", pause)
	if pause > 0 {
		throttle = time.Tick(time.Millisecond * time.Duration(pause))
	}
}

func throttleWait() {
	if throttle == nil {
		Debug.Println("Ignoring throttle")
		return
	}
	Debug.Println("Waiting throttle...")
	<-throttle
	Debug.Println("Waiting throttle done")
}


func main() {
	var query string
	var location string
	var category string
	var path_to_csvfile string
	var verbose bool
	var max_items int8
	var pause int64

	var SelaAvitoCmd = &cobra.Command{
		Use: "selavito",
		Short: "Утилита для парсинга объявлений (вместе с телефонными номерами) с сайта avito.ru",
		Example: "selavito -l moskva -q macbook --csv output.csv\niselavito -l sankt-peterburg -с rabota -q golang --csv output.csv",

		Run: func(cmd *cobra.Command, args []string) {
			InitLoggers(verbose)

			if query == "" || path_to_csvfile == "" {
				cmd.Help()
				return
			}

			throttleSet(pause)

			var page_url string
			counter := max_items
			items := make(chan *Item)
			save_wg := new(sync.WaitGroup)
			parse_wg := new(sync.WaitGroup)


			save_wg.Add(1)
			go saveToCSV(path_to_csvfile, items, save_wg)

			if category == "" {
				page_url = fmt.Sprintf("%s/%s?q=%s", BASE_URL, location, query)
			} else {
				page_url = fmt.Sprintf("%s/%s/%s?q=%s", BASE_URL, location, category, query)
			}

			// max_items == 0 - без ограничения
			for page_url != "" && (counter > 0 || max_items == 0) {
				Info.Println(page_url)

				throttleWait()

				doc, err := goquery.NewDocument(page_url)
				if err != nil {
					Error.Println(err)
					break
				}

				next_page_url, exists := doc.Find(".page-next").Find("a").First().Attr("href")
				if exists {
					next_page_url = fmt.Sprintf("%s%s", BASE_URL, next_page_url)
					Debug.Println("Next page:", next_page_url)
				}

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
	SelaAvitoCmd.Flags().StringVar(&path_to_csvfile, "csv", "",
		"Путь к csv файлу для сохранения данных")

	SelaAvitoCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Более подробный вывод в консоль")

	SelaAvitoCmd.Flags().Int8VarP(&max_items, "max", "m", 1,
		"Максимальное количество элементов для поиска (0 - без ограничения)")
	SelaAvitoCmd.Flags().Int64VarP(&pause, "pause", "p", 0,
		"Пауза между запросами (в микросекундах)")

	SelaAvitoCmd.Execute()

}

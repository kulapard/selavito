package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kulapard/selavito/Godeps/_workspace/src/github.com/PuerkitoBio/goquery"
	"github.com/kulapard/selavito/Godeps/_workspace/src/github.com/fatih/color"
	"github.com/kulapard/selavito/Godeps/_workspace/src/github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const BASE_URL string = "https://m.avito.ru"

var IPBanned error = errors.New("Ваш IP забанили!!")
var throttle <-chan time.Time

type Item struct {
	header   string
	location string
	url      string
	phone    string
}

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func Debug(format string, v ...interface{}) {
	DebugLogger.Println(color.GreenString(format, v...))
}

func Info(format string, v ...interface{}) {
	InfoLogger.Println(color.YellowString(format, v...))
}

func Error(format string, v ...interface{}) {
	ErrorLogger.Println(color.RedString(format, v...))
}

func InitLoggers(verbose bool) {
	var infoHandle, errorHandle, debugHandle io.Writer

	if verbose {
		debugHandle = os.Stdout
	} else {
		debugHandle = ioutil.Discard
	}

	infoHandle = os.Stdout
	errorHandle = os.Stderr

	DebugLogger = log.New(debugHandle, "", 0)
	InfoLogger = log.New(infoHandle, "", 0)
	ErrorLogger = log.New(errorHandle, "", 0)
}

// Достаёт телефонный номер из JSON по заданному URL
func getPhone(phone_url, referer string) (string, error) {
	Debug("Persing phone url: %s", phone_url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", phone_url, nil)
	req.Header.Add("referer", referer)

	throttleWait()
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Error("%s", err.Error())
		return "", err
	}
	if res.StatusCode == 403 {
		return "", IPBanned
	}

	phone_data := make(map[string]string)
	err = json.Unmarshal(body, &phone_data)
	if err != nil {
		Error("%s", err.Error())
		return "", err
	}

	Debug("Phone number: %s", phone_data["phone"])
	return phone_data["phone"], nil
}

func parseItem(item *Item, wg *sync.WaitGroup, items chan *Item) {
	throttleWait()

	doc, err := goquery.NewDocument(item.url)
	if err != nil {
		Error("%s", err.Error())
		wg.Done()
		return
	}

	item.location = doc.Find(".avito-address-text").First().Text()
	item.location = strings.TrimSpace(item.location)

	doc.Find(".action-show-number").Each(func(i int, s *goquery.Selection) {
		phone_url, exists := s.Attr("href")
		if exists {
			Debug("Found phone url: %s", phone_url)

			phone_url := strings.Join([]string{BASE_URL, phone_url, "?async"}, "")
			item.phone, err = getPhone(phone_url, item.url)
			if err != nil {
				Error("%s", err.Error())
			} else {
				items <- item
			}
		}
	})
	wg.Done()
}

func saveToCSV(path_to_csvfile string, items chan *Item, wg *sync.WaitGroup) {
	csvfile, err := os.Create(path_to_csvfile)

	if err != nil {
		Error("%s", err.Error())
		return
	}
	defer csvfile.Close()

	w := csv.NewWriter(csvfile)

	for item := range items {
		record := []string{item.header, item.location, item.phone, item.url}
		if err := w.Write(record); err != nil {
			Error("Не удалось записать в csv файл: %s", err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		Error("%s", err.Error())
	}

	wg.Done()
}

func throttleSet(pause int64) {
	Debug("Set throttle pause: %d ms", pause)
	if pause > 0 {
		throttle = time.Tick(time.Millisecond * time.Duration(pause))
	}
}

func throttleWait() {
	if throttle == nil {
		Debug("Ignoring throttle")
		return
	}
	Debug("Waiting throttle...")
	<-throttle
	Debug("Waiting throttle done")
}

func main() {
	var query string
	var location string
	var category string
	var path_to_csvfile string
	var verbose bool
	var max_items int64
	var pause int64

	var SelaAvitoCmd = &cobra.Command{
		Use:     "selavito",
		Short:   "Утилита для парсинга объявлений (вместе с телефонными номерами) с сайта avito.ru",
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

			items_done := 0

			// max_items == 0 - без ограничения
			for page_url != "" && (counter > 0 || max_items == 0) {
				Info("Парсинг страницы: %s", page_url)

				throttleWait()

				doc, err := goquery.NewDocument(page_url)
				if err != nil {
					Error(err.Error())
					break
				}

				next_page_url, exists := doc.Find(".page-next").Find("a").First().Attr("href")
				if exists {
					next_page_url = fmt.Sprintf("%s%s", BASE_URL, next_page_url)
					Info("Следующая страница: %s", next_page_url)
				}

				items_category := doc.Find(".nav-helper-header").First().Text()
				if items_category == "" {
					Error("Неверный формат страницы! Скорее всего ваш IP забанили!")
					break
				}
				items_count := doc.Find(".nav-helper-text").First().Text()
				items_category = strings.TrimSpace(items_category)
				items_count = strings.TrimSpace(items_count)
				if items_done == 0 {
					fmt.Println("Категория:", items_category)
					fmt.Println("Найдено объявлений:", items_count)
				} else {
					fmt.Printf("Процесс выполнения: %d/%s\n", items_done, items_count)
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
							items_done++
							Debug("%+v\n", item)
						} else {
							Error(".item-link not found")
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

	SelaAvitoCmd.Flags().Int64VarP(&max_items, "max", "m", 1,
		"Максимальное количество элементов для поиска (0 - без ограничения)")
	SelaAvitoCmd.Flags().Int64VarP(&pause, "pause", "p", 0,
		"Пауза между запросами (в микросекундах)")

	SelaAvitoCmd.Execute()

}

package routes

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"
)

type Person struct {
	Id         string
	Date       string
	User       string
	Department string
	Software   string
	Seats      string
	Amount     string
}

const ServletContextPath = "api"

func Run(ctx context.Context) {
	file, err := os.Open("data.csv")
	if err != nil {
		panic(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)
	// Create a new CSV reader
	reader := csv.NewReader(file)
	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	var people []Person
	for _, record := range records {
		id := record[0]
		date := record[1]
		user := record[2]
		department := record[3]
		software := record[4]
		seats := record[5]
		amount := record[6]
		person := Person{Id: id, Date: date, User: user, Department: department, Software: software, Seats: seats, Amount: amount}
		people = append(people, person)
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	engine := gin.Default()
	router := engine.Group(ServletContextPath)
	router.Use(gin.Recovery())
	httpServ := &http.Server{
		Addr:    ":" + "8081",
		Handler: engine,
	}
	expose(router, people)
	go func() {
		if err := httpServ.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalln("Could not start router", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	defer close(quit)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")
	_, cancel = context.WithTimeout(childCtx, 5*time.Second)
	defer cancel()
	if err := httpServ.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown")
	}
	log.Println("Server exiting")
}

func expose(router *gin.RouterGroup, people []Person) {

	router.GET("/total_items", getTotal(people))
	router.GET("/i/nth_most_total_item", soldItem(people))
	router.GET("/percentage_of_department_wise_sold_items", percentage(people))
	router.GET("/monthly_sales", monthly(people))
}
func getTotal(people []Person) gin.HandlerFunc {
	return func(c *gin.Context) {

		sum := 0
		for i, person := range people {
			if i >= 1 {
				month, err := strconv.Atoi(person.Date[5:7])
				if err != nil {
					panic(err)
				}
				dep := person.Department
				if month >= 7 && month <= 9 && dep == "Marketting" {
					seat, err := strconv.Atoi(person.Seats)
					if err != nil {
						panic(err)
					}
					sum = sum + seat
				}
			}
		}
		c.JSON(200, gin.H{
			"the total sum is ": sum,
		})
	}
}
func soldItem(people []Person) gin.HandlerFunc {
	return func(c *gin.Context) {
		hashmap := make(map[string]int)
		hashmap1 := make(map[string]int)
		for i, person := range people {
			if i >= 1 {
				month := person.Date[5:7]
				softwares := person.Software
				seat, err := strconv.Atoi(person.Seats)
				if err != nil {
					//panic(err)
					log.Fatalf("this is 2st error %v", err)
				}
				if month >= "10" && month <= "12" {
					_, ok := hashmap[softwares]
					if ok {
						hashmap[softwares] += seat
					} else {
						hashmap[softwares] = seat
					}
				}
				if month >= "04" && month <= "06" {
					_, ok := hashmap1[softwares]
					if ok {
						hashmap1[softwares] += seat
					} else {
						hashmap1[softwares] = seat
					}
				}
			}
		}
		mapArray1 := make([][2]string, len(hashmap))
		mapArray2 := make([][2]string, len(hashmap1))
		i := 0
		for key, value := range hashmap {
			mapArray1[i][0] = key
			mapArray1[i][1] = strconv.Itoa(value)
			i++
		}
		i = 0
		for key, value := range hashmap1 {
			mapArray2[i][0] = key
			mapArray2[i][1] = strconv.Itoa(value)
			i++
		}
		col := 1
		sort.Slice(mapArray1, func(i, j int) bool {
			return mapArray1[i][col] < mapArray1[j][col]
		})

		sort.Slice(mapArray2, func(i, j int) bool {
			return mapArray2[i][col] < mapArray2[j][col]
		})
		sz1 := len(mapArray1)
		sz2 := len(mapArray2)
		str1 := "2nd most sold item in terms of quantity sold in q4: " + mapArray1[sz1-2][0] + ", " + mapArray1[sz1-2][1] + "\n"
		str2 := "4th most sold item in terms of quantity sold in q2: " + mapArray2[sz2-4][0] + ", " + mapArray2[sz2-4][1]
		fmt.Println(str1 + str2)
		c.JSON(200, gin.H{
			"First":  str1,
			"second": str2,
		})
	}
}
func percentage(people []Person) gin.HandlerFunc {
	return func(c *gin.Context) {
		hashmap := make(map[string]int)
		sum := 0
		for i, person := range people {
			if i >= 1 {
				dep := person.Department
				seat, err := strconv.Atoi(person.Seats)
				if err != nil {
					panic(err)
				}
				sum = sum + seat
				if val, ok := hashmap[dep]; ok {
					seat += val
				}
				hashmap[dep] = seat
			}
		}
		var mapArray []map[string]interface{}
		for key, value := range hashmap {
			percentages := float64(value) * 100 / float64(sum)
			mapArray = append(mapArray, map[string]interface{}{
				"department": key,
				"percentage": percentages,
			})
		}
		c.JSON(200, gin.H{
			"percentage is ": mapArray,
		})
	}
}

func monthly(people []Person) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := make(map[string]map[string]map[string]float64)
		for i, entry := range people {
			if i >= 1 {
				date := entry.Date
				year := date[:4]
				month := date[5:7]
				software := entry.Software
				amountin := entry.Amount
				amount, err := strconv.ParseFloat(amountin, 64)
				if err != nil {
					panic(err)
				}
				if _, ok := results[software]; !ok {
					results[software] = make(map[string]map[string]float64)
				}
				if _, ok := results[software][year]; !ok {
					results[software][year] = make(map[string]float64)
				}
				results[software][year][month] += amount
			}
		}
		for outer, inner := range results {
			for innerkey, value := range inner {
				fmt.Println(outer)
				fmt.Println(innerkey)
				fmt.Println(value)
			}
		}
		c.JSON(200, results)
	}
}

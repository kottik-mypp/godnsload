package main

import (
	"fmt"
	"net"
	"time"
	"sync"
)

func main() {
	domain := "example.com"

	duration := 5 * time.Second
	numWorkers := 3

	successCount := 0
	errorCount := 0


	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(){
			timer := time.NewTimer(duration)
			fmt.Printf("Start %d\n", i)
			for {
				select {
				case <-timer.C:
					fmt.Printf("End %d\n", i)
					wg.Done()
					return
				default:
					_, err := net.LookupIP(domain)
					if err != nil {
						errorCount++
					} else {
						successCount++
					}
				}
			}
		}()
	}

	wg.Wait()

	fmt.Printf("Всего запросов: %d\n", successCount+errorCount)
	fmt.Printf("Успешных запросов: %d\n", successCount)
	fmt.Printf("Ошибочных запросов: %d\n", errorCount)
}

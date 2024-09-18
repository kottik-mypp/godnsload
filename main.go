package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/time/rate"
)

var (
	dnsServer  string // Адрес DNS-сервера (с портом)
	domain     string // Домен для запроса
	concurrent int    // Количество параллельных потоков
	rateLimit  int    // Количество запросов в секунду
	duration   int    // Длительность теста в секундах

	// Счётчики успешных и неуспешных запросов
	successCount uint64
	errorCount   uint64
)

func init() {
	// Чтение аргументов командной строки
	flag.StringVar(&dnsServer, "server", "192.168.88.1:53", "Адрес и порт DNS-сервера (например, 192.168.88.1:53)")
	flag.StringVar(&domain, "domain", "example.com.", "Домен для запроса")
	flag.IntVar(&concurrent, "concurrent", 100, "Количество параллельных потоков")
	flag.IntVar(&rateLimit, "rate", 50, "Количество запросов в секунду")
	flag.IntVar(&duration, "duration", 60, "Длительность теста в секундах")

	// Парсим аргументы
	flag.Parse()
}

func makeDNSRequest(domain, dnsServer string) error {
	// Создаём DNS сообщение
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeA)

	// Отправляем запрос на сервер
	c := new(dns.Client)
	_, _, err := c.Exchange(m, dnsServer)
	if err != nil {
		return err
	}
	return nil
}

func worker(id int, wg *sync.WaitGroup, limiter *rate.Limiter, ch chan struct{}) {
	defer wg.Done()
	for range ch {
		// Ограничиваем выполнение запросов с использованием лимитера
		err := limiter.Wait(context.Background())
		if err != nil {
			log.Printf("Worker %d: ошибка лимитера: %v", id, err)
			return
		}

		// Выполняем DNS запрос
		err = makeDNSRequest(domain, dnsServer)
		if err != nil {
			atomic.AddUint64(&errorCount, 1) // Увеличиваем счётчик ошибок
			log.Printf("Worker %d: ошибка запроса: %v", id, err)
		} else {
			atomic.AddUint64(&successCount, 1) // Увеличиваем счётчик успешных запросов
		}
	}
}

func main() {
	start := time.Now()

	var wg sync.WaitGroup
	ch := make(chan struct{})

	// Лимитер для ограничения запросов
	limiter := rate.NewLimiter(rate.Limit(rateLimit), 1) // rateLimit запросов в секунду

	// Запускаем воркеров для обработки запросов
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go worker(i, &wg, limiter, ch)
	}

	// Таймер для завершения теста после заданного времени
	timeout := time.After(time.Duration(duration) * time.Second)

	go func() {
		for {
			select {
			case ch <- struct{}{}: // Отправляем сигнал воркерам для выполнения запроса
			case <-timeout:
				fmt.Println("Тест завершён по тайм-ауту.")
				close(ch)
				return
			}
		}
	}()

	wg.Wait()

	totalDuration := time.Since(start)
	fmt.Printf("Все запросы выполнены за %v\n", totalDuration)
	fmt.Printf("Успешные запросы: %d\n", atomic.LoadUint64(&successCount))
	fmt.Printf("Ошибочные запросы: %d\n", atomic.LoadUint64(&errorCount))
}

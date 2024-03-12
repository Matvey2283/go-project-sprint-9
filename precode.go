package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch)
	var i int64
	for {
		i++
		select {
		case <-ctx.Done():
			return
		case ch <- i:
			fn(i)
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64, done <-chan struct{}) {
	defer close(out)
	for {
		select {
		case num, ok := <-in:
			if !ok {
				return
			}
			out <- num
			time.Sleep(1 * time.Millisecond) // Можно убрать это для ускорения обработки
		case <-done:
			return
		}
	}
}

func main() {
	chIn := make(chan int64)

	// Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	var mu sync.Mutex
	var wg sync.WaitGroup

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		mu.Lock()
		defer mu.Unlock()
		inputSum += i
		inputCount++
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	outs := make([]chan int64, NumOut)
	done := make(chan struct{}) // канал для оповещения горутин о завершении работы
	defer close(done)

	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		wg.Add(1)
		go func(in <-chan int64, out chan<- int64, amounts []int64) {
			defer wg.Done()
			for num := range in {
				out <- num
				amounts[i]++
			}
		}(chIn, outs[i], amounts)
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	go func() {
		wg.Wait()
		close(chOut)
	}()

	go func() {
		for _, in := range outs {
			for num := range in {
				chOut <- num
			}
		}
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	for num := range chOut {
		count++
		sum += num
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
}

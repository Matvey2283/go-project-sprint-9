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
	n := int64(1)
	for {
		select {
		case <-ctx.Done():
			close(ch) // закрываем канал после завершения работы генератора
			return
		default:
			fn(n)
			ch <- n
			n++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := range in {
		out <- i
	}
}

func main() {
	chIn := make(chan int64)

	// Создание контекста с таймаутом 1 секунда
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		inputSum += i
		inputCount++
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов

	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
	}

	var wg sync.WaitGroup
	wg.Add(NumOut)

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)

	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64)

	// Запускаем горутины Worker
	for i := 0; i < NumOut; i++ {
		go Worker(chIn, outs[i], &wg)
	}

	// Собираем числа из каналов outs
	go func() {
		wg.Wait()
		close(chOut)
	}()

	for i := 0; i < NumOut; i++ {
		go func(idx int) {
			for val := range outs[idx] {
				chOut <- val
				amounts[idx]++
			}
		}(i)
	}

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// Читаем числа из результирующего канала
	for val := range chOut {
		sum += val
		count++
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}

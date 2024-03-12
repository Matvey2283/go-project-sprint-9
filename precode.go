package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// вызывает переданную функцию для каждого из
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	var val int64 = 1
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ch <- val
			fn(val)
			val++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	defer close(out)
	for {
		v, ok := <-in
		if !ok {
			break
		}
		out <- v
		time.Sleep(time.Millisecond)
	}
}

var mu sync.Mutex

func main() {
	chIn := make(chan int64)
	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		mu.Lock()
		inputSum += i
		inputCount++
		mu.Unlock()
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	outs := make([]chan int64, NumOut)
	for i := range outs {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	chOut := make(chan int64)
	go func() { // сливаем все outs в один канал chOut
		defer close(chOut)
		wg := sync.WaitGroup{}
		wg.Add(NumOut)
		for _, out := range outs {
			go func(out <-chan int64) {
				for v := range out {
					chOut <- v
				}
				wg.Done()
			}(out)
		}
		wg.Wait()
	}()

	var wg sync.WaitGroup
	// 4. Собираем числа из каналов outs
	for i := 0; i < NumOut; i++ {
		wg.Add(1)
		go func(in <-chan int64, i int64) {
			defer wg.Done()
			for n := range in {
				amounts[i]++
				chOut <- n
			}
		}(outs[i], int64(i))
	}

	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		close(chIn)
	}()

	var count int64 // количество числе результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	for v := range chOut {
		count++
		sum += v
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
}

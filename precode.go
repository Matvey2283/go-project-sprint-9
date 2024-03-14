package main

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Generator генерирует последовательность чисел 1, 2, 3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch) // закрываем канал при выходе из функции
	i := int64(1)
	for {
		select {
		case <-ctx.Done():
			return // выходим из функции, если контекст отменен
		case ch <- i:
			fn(i)
			i++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64, wgStat *sync.WaitGroup) {
	defer wgStat.Done() // уменьшаем счетчик группы, когда завершаем работу
	defer close(out)    // закрываем канал при выходе из функции
	for v := range in {
		out <- v
	}
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел
	var mutex sync.Mutex

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		mutex.Lock()
		defer mutex.Unlock()
		inputSum += i
		inputCount++
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов

	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
	}

	// wgStat используется для отслеживания завершения всех горутин Worker
	var wgStat sync.WaitGroup
	wgStat.Add(NumOut)

	// запускаем обработчики чисел
	for i := 0; i < NumOut; i++ {
		go Worker(chIn, outs[i], &wgStat)
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)

	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// 4. Собираем числа из каналов outs
	for i, out := range outs {
		wg.Add(1)
		go func(in <-chan int64, i int) {
			defer wg.Done()
			for v := range in {
				mutex.Lock()
				amounts[i]++
				mutex.Unlock()
				chOut <- v
			}
		}(out, i)
	}

	go func() {
		// ждём завершения работы всех горутин для сбора статистики
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	for v := range chOut {
		count++
		sum += v
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

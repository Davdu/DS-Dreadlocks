package main

import (
	"fmt"
	"time"
)

func main() {

	ch11 := make(chan int)
	ch12 := make(chan int)
	ch22 := make(chan int)
	ch23 := make(chan int)
	ch33 := make(chan int)
	ch34 := make(chan int)
	ch44 := make(chan int)
	ch45 := make(chan int)
	ch55 := make(chan int)
	ch51 := make(chan int)

	go fork(1, ch11, ch12)
	go fork(2, ch22, ch23)
	go fork(3, ch33, ch34)
	go fork(4, ch44, ch45)
	go fork(5, ch55, ch51)

	go eater(1, ch51, ch11)
	go eater(2, ch12, ch22)
	go eater(3, ch23, ch33)
	go eater(4, ch34, ch44)
	go eater(5, ch45, ch55)

	time.Sleep(10000 * time.Millisecond)
}

func fork(number int, ch1, ch2 chan int) {
	var x int = 1

	if number%2 == 0 {
		x = <-ch1
	} else {
		x = <-ch2
	}

	fmt.Println(x)

	/*for i := 0; i < 1000000000; i++ {
		ch1 <- 5
		ch2 <- 10
	}*/
}

func eater(number int, ch1, ch2 chan int) {

	var y int = 0

	select {
	case ch1 <- y:
		y++
	case ch2 <- y:
		y++
	default:
		fmt.Println("eater ", number, " not consumed")
	}

	if y == 1 {
		fmt.Println("Eater ", number, "Channel 1")
	}
}

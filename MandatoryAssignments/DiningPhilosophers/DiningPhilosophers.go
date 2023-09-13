package main

import (
	"fmt"
	"time"
)

func main() {

	ch1 := make(chan bool)
	ch2 := make(chan bool)
	ch3 := make(chan bool)
	ch4 := make(chan bool)
	ch5 := make(chan bool)

	go fork(ch1)
	go fork(ch2)
	go fork(ch3)
	go fork(ch4)
	go fork(ch5)

	go eater(1, ch5, ch1)
	go eater(2, ch1, ch2)
	go eater(3, ch2, ch3)
	go eater(4, ch3, ch4)
	go eater(5, ch4, ch5)

	time.Sleep(100000000 * time.Millisecond)
}

func fork(ch chan bool) {
	ch <- true
}

func eater(number int, ch1, ch2 chan bool) {

	var y int = 0

	for {

		// Avoids deadlock
		if number%2 == 0 {
			<-ch1
			<-ch2
		} else {
			<-ch2
			<-ch1
		}

		y++

		fmt.Printf("Philosopher no: %v, State: Eating, Count: %v\n", number, y)

		ch1 <- true
		ch2 <- true

		fmt.Printf("Philosopher no: %v, State: Thinking\n", number)
	}

}

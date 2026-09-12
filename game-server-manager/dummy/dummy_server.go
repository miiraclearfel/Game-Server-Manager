package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting Dummy Game Server...")
	for {
		time.Sleep(2 * time.Second)
		fmt.Println("Dummy Game Server is running...")
	}
}

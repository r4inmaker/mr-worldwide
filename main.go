package main

import "fmt"

func print(args ...interface{}) {
	fmt.Println(args...)
}

func main() {
	print("Hello Mr!")
}

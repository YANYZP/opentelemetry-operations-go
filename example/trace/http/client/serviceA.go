package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for true {
		fmt.Println("Enter your ingredient: \n (\"bye\" to quit)")
		scanner.Scan()
		ingredientName := scanner.Text()
		if ingredientName == "bye" {
			return
		}
		fmt.Println("Your text was: ", ingredientName)
	}
}

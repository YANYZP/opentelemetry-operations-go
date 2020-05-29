package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("vendor.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var vendorOfItemMap = make(map[string][]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		itemVendorInfo := scanner.Text()
		words := strings.Fields(itemVendorInfo)
		fmt.Println(itemVendorInfo)
		for i := 1; i < len(words); i++ {
			vendorOfItemMap[words[0]] = append(vendorOfItemMap[words[0]], words[i])
		}
	}

	for k, v := range vendorOfItemMap {
		for _, vv := range v {
			fmt.Println(k, vv)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	file, err = os.Open("price.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var itemPriceMap = make(map[string]map[string]string)

	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		priceInfo := scanner.Text()
		words := strings.Fields(priceInfo)
		fmt.Println(priceInfo)
		if len(words) != 3 {
			fmt.Println("Wrong format")
			continue
		}
		if itemPriceMap[words[0]] == nil {
			itemPriceMap[words[0]] = make(map[string]string)
		}
		itemPriceMap[words[0]][words[1]] = words[2]
	}

	for k, v := range itemPriceMap {
		for _, vv := range v {
			fmt.Println(k, v, vv)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}

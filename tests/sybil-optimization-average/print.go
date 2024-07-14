package main

import (
	"fmt"
	"reflect"
)

func PrintUsefulCpl(nodesPerCpl interface{}) {
	var minCpl, maxCpl int
	nodes := reflect.ValueOf(nodesPerCpl)

	if nodes.Kind() == reflect.Slice || nodes.Kind() == reflect.Array {
		var count int64

		for i := 0; i < nodes.Len(); i++ {
			if count == 0 && nodes.Index(i).Int() != 0 {
				minCpl = i
			}

			if nodes.Index(i).Int() != 0 {
				maxCpl = i
			}

			count += nodes.Index(i).Int()
		}

		fmt.Printf("                 ")
		for i := minCpl; i <= maxCpl; i++ {
			fmt.Printf("%4d", i)
		}
		fmt.Println()

		fmt.Printf("Nodes per CPL : ")
		fmt.Printf("[")
		for i := minCpl; i <= maxCpl; i++ {
			fmt.Printf(" %3d", nodes.Index(i).Int())
		}
		fmt.Printf(" ]\n")
	}
}

func printArraysAsCsv(input ...interface{}) {
	// fmt.Printf("[")
	arraySlice := reflect.ValueOf(input)
	var array []reflect.Value

	if arraySlice.Kind() == reflect.Slice {
		for i := 0; i < arraySlice.Len(); i++ {
			value := reflect.ValueOf(arraySlice.Index(i).Interface())

			if value.Kind() == reflect.Slice {
				array = append(array, value)
			}
		}
	}

	for j := 0; j < array[0].Len(); j++ {
		for i := 0; i < len(array); i++ {
			value := array[i].Index(j)

			if i == 0 {
				fmt.Print("", value, "")
				continue
			}

			fmt.Print(", ", value)
		}

		fmt.Println()
	}

	// fmt.Printf("]\n")
}

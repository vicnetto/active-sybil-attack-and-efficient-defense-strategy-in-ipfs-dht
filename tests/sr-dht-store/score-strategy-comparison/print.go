package main

import (
	"fmt"
	"reflect"
)

func printGlobalStatus(data PriorityTestLog) {
	quantityOfResults := float64(len(data.score))
	log.Info.Printf("  Average score (min/max): %f (%f/%f)", data.scoreSum/quantityOfResults, data.scoreLow, data.scoreHigh)
	log.Info.Printf("  Average Kl (min/max): %f (%f/%f)", data.klSum/quantityOfResults, data.klLow, data.klHigh)
	log.Info.Printf("  Average quantity of Sybils (min/max): %f (%d/%d)", float64(data.sybilSum)/quantityOfResults, data.sybilLow, data.sybilHigh)
	log.Info.Printf("  Average of closer sybils than all reliable (min/max): %f (%d/%d)\n", float64(data.closerThenAllReliableSum)/quantityOfResults, data.closerSybilsLow, data.closerSybilsHigh)
	log.Info.Printf("  Closest is a sybil in: %d%%", int(float64(data.closestIsSybilSum)/quantityOfResults*100))
}

func printCurrentStatus(currentPosition int, data PriorityTestLog) {
	quantityOfResults := float64(currentPosition) + 1

	log.Info.Printf("  Score (average): %f (%f)\n", data.score[currentPosition], data.scoreSum/quantityOfResults)
	log.Info.Printf("  Kl (average): %f (%f)\n", data.kl[currentPosition], data.klSum/quantityOfResults)
	log.Info.Printf("  Sybils (average): %d (%f)\n", data.sybil[currentPosition], float64(data.sybilSum)/quantityOfResults)
	log.Info.Printf("  Closest is sybil (average): %t (%d%%)\n", data.closestIsSybil[currentPosition],
		int(float64(data.closestIsSybilSum)/quantityOfResults*100))
	log.Info.Printf("  Closer than all reliable (average): %d (%f)\n",
		data.closerThanAllReliable[currentPosition],
		float64(data.closerThenAllReliableSum)/quantityOfResults)
}

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

		var cplString string
		for i := minCpl; i <= maxCpl; i++ {
			cplString += fmt.Sprintf("%4d", i)
		}
		log.Info.Printf("            CPL:  %s", cplString)

		var sybilString string
		for i := minCpl; i <= maxCpl; i++ {
			sybilString += fmt.Sprintf(" %3d", nodes.Index(i).Int())
		}
		log.Info.Printf("  Nodes per CPL: [%s ]", sybilString)
	}
}

func printArraysAsCsv(input ...interface{}) {
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
		var valueString string

		for i := 0; i < len(array); i++ {
			value := array[i].Index(j)

			if i == 0 {
				valueString += fmt.Sprint(value)
				continue
			}

			valueString += fmt.Sprint(", ", value)
		}

		fmt.Println(valueString)
	}
}

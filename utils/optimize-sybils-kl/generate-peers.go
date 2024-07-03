package main

// import (
// 	"fmt"
// 	"os"
// 	"os/exec"
// )
//
// var GeneratePidPath = "../generate_pid/generate_pid"
//
// func GeneratePidAccordingToSybils(sybils [MaxCplProbabilitySize]int, cid string) {
//
// 	fmt.Println()
// 	cmd := exec.Command(GeneratePidPath)
// 	cmd.Args = append(cmd.Args, "-byCpl")
// 	cmd.Args = append(cmd.Args, "-cid", cid)
//
// 	for i, quantity := range sybils {
// 		if quantity != 0 {
// 			cmd.Args = append(cmd.Args, fmt.Sprintf("-%d", i))
// 			cmd.Args = append(cmd.Args, fmt.Sprintf("%d", quantity))
// 		}
// 	}
//
// 	cmd.Stdout = os.Stdout
//
// 	if err := cmd.Run(); err != nil {
// 		fmt.Println("Generate PID failed!")
// 		fmt.Println(err)
// 	}
// }

package main

import (
	"crawler"
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [url]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	inputPtr := flag.String("input", "", "path to saved matrix")
	outputPtr := flag.String("output", "", "path to save matrix")
	flag.Parse()

	args := flag.Args()

	opts := crawler.NewOptions()
	opts.SameHostOnly = true
	c := crawler.NewCrawler(opts)

	if len(args) < 1 && *inputPtr == "" {
		fmt.Println("URL or input file required")
		os.Exit(1)
	}

	var outputf *os.File

	if *outputPtr == "" {
		outputf = nil
	} else {
		outputPath := *outputPtr
		var outputerr error
		outputf, outputerr = os.Create(outputPath)
		if outputerr != nil {
			panic(outputerr)
		}
		defer outputf.Close()
	}



	if len(args) >= 1 {
		c.Run(flag.Arg(0), outputf)
	} else {
		inputPath := *inputPtr
		inputf, inputerr := os.Open(inputPath)
		if inputerr != nil {
			panic(inputerr)
		}
		defer outputf.Close()
		c.ParseMatrix(inputf)
	}

}

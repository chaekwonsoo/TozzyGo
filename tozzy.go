package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"
)

func tozzy() {
	fmt.Println("//=========== Welcome to tozzy world! ===========")
	flagSet()
	
	readTozzyDesc()
}

func flagSet() { // DOTO - make them reality
	flag.String("word", "foo", "a string")
	flag.Int("numb", 42, "an int")
	flag.Bool("fork", false, "a bool")
	flag.Parse()
}

func readTozzyDesc() {
	if !hasCmlArg() {
		readAndEchoStdin()
	} else {
		parseTozzyFile()
	}
}

func hasCmlArg() bool {
	if flag.Arg(0) == "" {
		return false
	}

	return true
}

func readAndEchoStdin() {
	const waitTime = 10
	lines := scanForStdin()

	for {
		select {
		case line := <-lines: // line is []byte
			fmt.Println(line)
		case <-time.After(time.Second * time.Duration(waitTime)):
			// for each line wait for <waitTime> secs
			fmt.Println(waitTime, "secs expired for one line input... exiting")
			return
		}
	}
}

// get tozzy description from stdin
func scanForStdin() chan string {
	lines := make(chan string)
	go func() {
		for {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				lines <- scanner.Text()
			}
		}
	}()
	return lines
}

func parseTozzyFile() {
	lines := readLinesFromFile(flag.Arg(0))
	wholefile := accumulateReceivedBytes(lines)

	var builtins = map[string]interface{}{
		"printf": fmt.Printf,
	}

	_, err2 := Parse("TOZZY", string(wholefile), "%%", "%%", builtins)
	if err2 != nil {
	}
}

func readLinesFromFile(fname string) chan []byte {
	lines := make(chan []byte)
	go func() {
		var fin *os.File
		var err error
		if fin, err = os.Open(fname); err != nil {
			_, file, line, _ := runtime.Caller(0)
			log.Fatal("INPUT FILE READ ERROR", file, " ", line+1)
		}
		r := bufio.NewReader(fin)
		
		for {
			line, prefix, err := r.ReadLine()
			if prefix {
				log.Fatal("prefex: line too long")
			}
			if err == io.EOF {
				fmt.Println(string(line))
				lines <- line
				break
			} else if err != nil {
				log.Fatal("error occured during ReadLine")
			}
			lines <- line
		}
		close(lines)
	}()

	return lines
}

func accumulateReceivedBytes(lines <-chan []byte) []byte {
	wholefile := make([]byte, 0)

	for {
		if line, ok := <-lines; ok { // line is []byte
			//@@ fmt.Println(line)
			wholefile = append(wholefile, line...)
			wholefile = append(wholefile, '\n')
			//wholefile = append(wholefile, "\n"...) // this works also
		} else {
			break
		}
	}

	return wholefile
}

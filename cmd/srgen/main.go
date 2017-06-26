//    Copyright 2017 drillbits
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/drillbits/srgen"
)

var (
	output = flag.String("o", "services.go", "write the generated output to the named file, instead of the default name")
)

func usage(w io.Writer) func() {
	return func() {
		fmt.Fprint(w, `usage: srgen gofiles...

Generate the service registry by tagged interfaces.

  -o file
    write the generated output to the named file,
    instead of the default name 'services.go'.

`)
		os.Exit(2)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("srgen: ")
	flag.Usage = usage(os.Stderr)
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}

	err := srgen.Generate(args, *output)
	if err != nil {
		log.Fatalf("failed to generate: %s", err)
	}
}
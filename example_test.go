// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng_test

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpatheng"
)

func Example() {
	str := `
	<developer>
		<name>Santhosh Kumar Tekuri</name>
		<email>santhosh.tekuri@gmail.com</email>
	</developer>
	`
	doc, err := dom.Unmarshal(xml.NewDecoder(strings.NewReader(str)))
	if err != nil {
		fmt.Println(err)
		return
	}

	xpath, err := new(xpatheng.Compiler).Compile("/developer/name")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", xpath, xpath.Returns())

	result, err := xpath.EvalString(doc, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Result: %s", result)
	// Output:
	// xpath /developer/name returns value of type node-set
	// Result: Santhosh Kumar Tekuri
}

func ExampleVariableMap() {
	uri := "www.jroller.com/santhosh/"

	compiler := &xpatheng.Compiler{
		NS: map[string]string{
			"ns": uri,
		},
	}
	xpath, err := compiler.Compile("$v1 + $v2 * $ns:v3 - $ns:v4")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", xpath, xpath.Returns())

	result, err := xpath.EvalNumber(nil, xpatheng.VariableMap{
		"v1": float64(2),
		"v2": float64(3),
		"{www.jroller.com/santhosh/}v3": float64(4),
		xpatheng.ClarkName(uri, "v4"):   float64(1),
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Result: %.2f", result)
	// Output:
	// xpath $v1 + $v2 * $ns:v3 - $ns:v4 returns value of type number
	// Result: 13.00
}

func ExampleFunctionMap() {
	join := func(args []interface{}) interface{} {
		sep := args[0].(string)
		var a []string
		for _, v := range args[1:] {
			a = append(a, v.(string))
		}
		return strings.Join(a, sep)
	}

	uri := "www.jroller.com/santhosh/"

	compiler := &xpatheng.Compiler{
		NS: map[string]string{
			"x": uri,
		},
		Functions: xpatheng.FunctionMap{
			"{www.jroller.com/santhosh/}join": &xpatheng.Function{
				Returns: xpatheng.String,
				Args: xpatheng.Args{
					xpatheng.Mandatory(xpatheng.String),
					xpatheng.Variadic(xpatheng.String),
				},
				Compile: xpatheng.CompileFunc(join),
			},
		},
	}
	xpath, err := compiler.Compile("x:join(':', 'one', 'two', 'three')")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", xpath, xpath.Returns())

	result, err := xpath.EvalString(nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Result: %s", result)
	// Output:
	// xpath x:join(':', 'one', 'two', 'three') returns value of type string
	// Result: one:two:three
}

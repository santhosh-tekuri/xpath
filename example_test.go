// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpath_test

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpath"
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

	expr, err := new(xpath.Compiler).Compile("/developer/name")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", expr, expr.Returns())

	result, err := expr.EvalString(doc, nil)
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

	compiler := &xpath.Compiler{
		Namespaces: map[string]string{
			"ns": uri,
		},
	}
	expr, err := compiler.Compile("$v1 + $v2 * $ns:v3 - $ns:v4")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", expr, expr.Returns())

	result, err := expr.EvalNumber(nil, xpath.VariableMap{
		"v1": float64(2),
		"v2": float64(3),
		"{www.jroller.com/santhosh/}v3": float64(4),
		xpath.ClarkName(uri, "v4"):      float64(1),
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

	compiler := &xpath.Compiler{
		Namespaces: map[string]string{
			"x": uri,
		},
		Functions: xpath.FunctionMap{
			"{www.jroller.com/santhosh/}join": &xpath.Function{
				Returns: xpath.String,
				Args: xpath.Args{
					xpath.Mandatory(xpath.String),
					xpath.Variadic(xpath.String),
				},
				Compile: xpath.CompileFunc(join),
			},
		},
	}
	expr, err := compiler.Compile("x:join(':', 'one', 'two', 'three')")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", expr, expr.Returns())

	result, err := expr.EvalString(nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Result: %s", result)
	// Output:
	// xpath x:join(':', 'one', 'two', 'three') returns value of type string
	// Result: one:two:three
}

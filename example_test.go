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

func Example_Variables() {
	uri := "www.jroller.com/santhosh/"

	compiler := &xpatheng.Compiler{
		NS: map[string]string{
			"ns": uri,
		},
	}
	xpath, err := compiler.Compile("$v1 + $v2 * $ns:v3")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("xpath %v returns value of type %v\n", xpath, xpath.Returns())

	result, err := xpath.EvalNumber(nil, xpatheng.VariableMap{
		"v1": float64(2),
		"v2": float64(3),
		xpatheng.ClarkName(uri, "v3"): float64(4),
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Result: %.2f", result)
	// Output:
	// xpath $v1 + $v2 * $ns:v3 returns value of type number
	// Result: 14.00
}

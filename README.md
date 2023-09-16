# 📦 Pack

[![GoDoc](https://godoc.org/github.com/NublyBR/go-pack?status.png)](http://godoc.org/github.com/NublyBR/go-pack)
[![Go Report Card](https://goreportcard.com/badge/github.com/NublyBR/go-pack)](https://goreportcard.com/report/github.com/NublyBR/go-pack)

A Go lib for packing and unpacking Go types into binary data, for easy storage and network streaming.

# ⚡️ Basic Usage

```go
package main

import (
	"fmt"
    
	"github.com/NublyBR/go-pack"
)

type CustomType struct {
	String string
    Number int
}

func main() {
	value := CustomType{
    	String: "Hello, World!",
        Number: 123,
    }
    
	data, err := pack.Marshal(value)
    if err != nil {
    	panic(err)
    }

    fmt.Printf("Value encoded as bytes: %q\n", data)
    
    var decoded CustomType
    
	err = pack.Unmarshal(&decoded)
    if err != nil {
    	panic(err)
    }
    
    fmt.Printf("Value decoded from bytes: %+v\n", decoded)
}

```

# ⚙️ Installation

```
go get -u github.com/NublyBR/go-pack
```
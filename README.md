# 📦 Pack

[![Build Status](https://github.com/NublyBR/go-pack/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/NublyBR/go-pack/actions/workflows/go.yml/badge.svg?query=branch%3Amain)
[![GoDoc](https://godoc.org/github.com/NublyBR/go-pack?status.png)](http://godoc.org/github.com/NublyBR/go-pack)
[![Go Report Card](https://goreportcard.com/badge/github.com/NublyBR/go-pack?)](https://goreportcard.com/report/github.com/NublyBR/go-pack)

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

# 📈 Benchmarks

```
$ go test -benchmem -bench=.

goos: linux
goarch: amd64
pkg: github.com/NublyBR/go-pack
cpu: Intel(R) Core(TM) i5-9600K CPU @ 3.70GHz
BenchmarkPacker-6         674068              1619 ns/op            1000 B/op         23 allocs/op
BenchmarkUnpacker-6       526233              2101 ns/op             656 B/op         27 allocs/op
PASS
ok      github.com/NublyBR/go-pack      2.669s
```

The benchmarks are executed by packing/unpacking the following struct:

```go
{
    String: "Hello, World!",
    Int:    1337_1337,
    Float:  1337.1337,
    Slice:  []any{"Hello, World!", 1337_1337, 1337.1337},
    Map: map[string]any{
        "abc": 1337_1337,
        "def": 1337.1337,
    },
}
```

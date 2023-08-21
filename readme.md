# Reverse Scanner

![Build Status](https://github.com/vallerion/rscanner/actions/workflows/go.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/vallerion/rscanner.svg)](https://pkg.go.dev/github.com/vallerion/rscanner)
[![Go Report Card](https://goreportcard.com/badge/github.com/vallerion/rscanner)](https://goreportcard.com/report/github.com/vallerion/rscanner)
[![codecov](https://codecov.io/gh/vallerion/rscanner/branch/master/graph/badge.svg)](https://codecov.io/gh/vallerion/rscanner)

Have you ever found yourself in a situation where you need to read a file or another readable source from the end to the
start? While `bufio.Scanner` only supports reading from the start, `rscanner.Scanner` offers the capability to read from
the end.

`rscanner.Scanner` is an exceptionally efficient and thoroughly tested reverse scanner, even in production.

The `rscanner` package intentionally maintains the same interface as `bufio.Scanner` for reading,
including `scanner.Scan()`,`scanner.Bytes()`, and `scanner.Text()`. However, `rscanner` introduces some slight
differences to offer enhanced configurability, such as `scanner.Buffer()`, `scanner.MaxTokenSize()`,
and `scanner.MaxConsecutiveEmptyReads()`.

To achieve superior performance, it minimizes memory allocation and reuses the same block of memory for multiple reads.

## Install

```shell
go get -t github.com/vallerion/rscanner
```

## Usage

### Simple
```go
const text = "run\nforrest\nrun"
sc := rscanner.NewScanner(strings.NewReader(text), int64(len(text)))
for sc.Scan() {
    log.Println(sc.Text())
}
if sc.Err() != nil {
    log.Fatalf("woops! err: %v", sc.Err())
}
```

### File
```go
f, err := os.Open("service.log")
if err != nil {
    log.Fatalln(err)
}
defer f.Close()

fs, err := f.Stat()
if err != nil {
    log.Fatalln(err)
}

sc := rscanner.NewScanner(f, fs.Size())
for sc.Scan() {
    log.Println(sc.Text())
}

if sc.Err() != nil {
    log.Fatalf("woops! err: %v", sc.Err())
}

log.Println("File is finished!")
```
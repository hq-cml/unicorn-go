package main

import (
    "testing"
    "runtime"
)

func TestStart(t *testing.T) {
    runtime.GOMAXPROCS(runtime.NumCPU())


}

#!/bin/bash
go test .
go test ./testdata
rm -f testdata/services.go
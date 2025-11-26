#!/usr/bin/env bash
go clean
export GOEXPERIMENT=greenteagc
go build -mod=mod --trimpath -o sandwich .
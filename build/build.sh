#!/usr/bin/env bash
go clean
go build -mod=mod --trimpath -o sandwich .
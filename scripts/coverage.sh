#!/bin/bash

go test -v -coverprofile c.out ./...
go tool cover -html=c.out
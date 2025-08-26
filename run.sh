#!/opt/homebrew/bin/bash

cd frontend && npm run build && cd ..
go build -o go-rest main.go
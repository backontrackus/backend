#!/bin/bash
cd /home/lukeo/backontrack/
go install
go run main.go serve --http 0.0.0.0:45097

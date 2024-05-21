#!/bin/bash
@echo off

squid -k kill

go run ./main.go

service squid start
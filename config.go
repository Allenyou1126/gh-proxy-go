package main

import (
	"github.com/joho/godotenv"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

var USE_JSDELIVR_AS_MIRROR_FOR_BRANCHES = false
var SIZE_LIMIT int64 = 1024 * 1024 * 1024 * 999

var HOST = "127.0.0.1"
var PORT = 80

var WHITE_LIST = make([][]string, 0)
var BLACK_LIST = make([][]string, 0)
var PASS_LIST = make([][]string, 0)

var CHUNK_SIZE = 1024 * 10

func parseList(input string) [][]string {
	var result [][]string
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "/")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		result = append(result, parts)
	}
	return result
}

func checkList(m []string, list [][]string) bool {
	for _, rule := range list {
		if matchRule(m, rule) {
			return true
		}
	}
	return false
}

func checkWhiteList(m []string) bool {
	if len(WHITE_LIST) == 0 {
		return true
	}
	return checkList(m, WHITE_LIST)
}

func checkBlackList(m []string) bool {
	return checkList(m, BLACK_LIST)
}

func checkPassList(m []string) bool {
	return checkList(m, PASS_LIST)
}

func init() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		slog.Error(err.Error())
		os.Exit(1)
	}
	if v, exist := os.LookupEnv("SERVER_HOST"); exist {
		HOST = v
	}
	if v, exist := os.LookupEnv("SERVER_PORT"); exist {
		v, err := strconv.Atoi(v)
		if err != nil {
			slog.Error("Invalid environment variable: SERVER_PORT")
			os.Exit(1)
		}
		PORT = v
	}
	if v, exist := os.LookupEnv("WHITE_LIST"); exist {
		WHITE_LIST = parseList(v)
	}
	if v, exist := os.LookupEnv("BLACK_LIST"); exist {
		BLACK_LIST = parseList(v)
	}
	if v, exist := os.LookupEnv("PASS_LIST"); exist {
		PASS_LIST = parseList(v)
	}
	if v, exist := os.LookupEnv("JSDELIVER_MIRROR"); exist {
		USE_JSDELIVER_AS_MIRROR_FOR_BRANCHES = v == "1" || v == "true" || v == "True" || v == "TRUE"
	}
	if v, exist := os.LookupEnv("SIZE_LIMIT"); exist {
		n, err := strconv.Atoi(v)
		if err != nil {
			c := int64(1)
			ns := v
			if strings.HasSuffix(v, "G") {
				c = 1024 * 1024 * 1024
				ns = v[0 : len(v)-1]
			} else if strings.HasSuffix(v, "M") {
				c = 1024 * 1024
				ns = v[0 : len(v)-1]
			} else if strings.HasSuffix(v, "K") {
				c = 1024
				ns = v[0 : len(v)-1]
			}
			n, err := strconv.Atoi(ns)
			if err != nil {
				slog.Error("Invalid environment variable: SIZE_LIMIT")
				os.Exit(1)
			}
			SIZE_LIMIT = int64(n) * c
		} else {
			SIZE_LIMIT = int64(n)
		}
	}
}

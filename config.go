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

var DEBUG_MODE = false

var CHUNK_SIZE = 1024 * 10

func parseList(input string) [][]string {
	slog.Debug("Parsing rule lists.", "input", input)
	var result [][]string
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		slog.Debug("Parsing line.", "current_line", line)
		parts := strings.Split(line, "/")
		slog.Debug("Splitting line.", "parts", parts)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		result = append(result, parts)
	}
	slog.Debug("Parsing succeeded.", "result", result)
	return result
}

func matchRule(m, rule []string) bool {
	slog.Debug("Matching rules for ", "m", m, "rule", rule)
	for i, part := range rule {
		if i >= len(m) {
			slog.Debug("Not matched.")
			return false
		}
		if part != "*" && m[i] != part {
			slog.Debug("No wildcard found and not matched.")
			return false
		}
	}
	slog.Debug("Matched.")
	return true
}

func checkList(m []string, list [][]string) bool {
	slog.Debug("Matching rules", "m", m, "list", list)
	for _, rule := range list {
		if matchRule(m, rule) {
			return true
		}
	}
	slog.Debug("No rule matched.")
	return false
}

func checkWhiteList(m []string) bool {
	slog.Debug("Checking whitelist.", "m", m)
	if len(WHITE_LIST) == 0 {
		slog.Debug("WhiteList is empty, ignore.")
		return true
	}
	return checkList(m, WHITE_LIST)
}

func checkBlackList(m []string) bool {
	slog.Debug("Checking blacklist.", "m", m)
	return checkList(m, BLACK_LIST)
}

func checkPassList(m []string) bool {
	slog.Debug("Checking passlist.", "m", m)
	return checkList(m, PASS_LIST)
}

func loadEnv() {
	slog.Debug("Loading dotenv.")
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		slog.Error("Load .env file failed.", "err", err)
		os.Exit(1)
	}
	slog.Debug("Environment variables loaded successfully.", "env", os.Environ())
	if !DEBUG_MODE {
		slog.Debug("Loading DEBUG_MODE.")
		if v, exist := os.LookupEnv("DEBUG_MODE"); exist {
			slog.Debug("Found DEBUG_MODE in environment.", "origin", v)
			DEBUG_MODE = v == "1" || v == "true" || v == "True" || v == "TRUE"
		}
		slog.Debug("DEBUG_MODE initialized.", "DEBUG_MODE", DEBUG_MODE)
		if DEBUG_MODE {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
			loadEnv()
			return
		}
	}
	slog.Debug("Loading SERVER_HOST.")
	if v, exist := os.LookupEnv("SERVER_HOST"); exist {
		slog.Debug("Found SERVER_HOST in environment.", "origin", v)
		HOST = v
	}
	slog.Debug("SERVER_HOST initialized.", "SERVER_HOST", HOST)
	slog.Debug("Loading SERVER_PORT.")
	if v, exist := os.LookupEnv("SERVER_PORT"); exist {
		slog.Debug("Found SERVER_PORT in environment.", "origin", v)
		v, err := strconv.Atoi(v)
		if err != nil {
			slog.Error("Invalid environment variable: SERVER_PORT", "err", err)
			os.Exit(1)
		}
		PORT = v
	}
	slog.Debug("SERVER_PORT initialized.", "SERVER_PORT", PORT)
	slog.Debug("Loading WHITE_LIST.")
	if v, exist := os.LookupEnv("WHITE_LIST"); exist {
		slog.Debug("Found WHITE_LIST in environment.", "origin", v)
		WHITE_LIST = parseList(v)
	}
	slog.Debug("WHITE_LIST initialized.", "WHITE_LIST", WHITE_LIST)
	slog.Debug("Loading BLACK_LIST.")
	if v, exist := os.LookupEnv("BLACK_LIST"); exist {
		slog.Debug("Found BLACK_LIST in environment.", "origin", v)
		BLACK_LIST = parseList(v)
	}
	slog.Debug("BLACK_LIST initialized.", "BLACK_LIST", BLACK_LIST)
	slog.Debug("Loading PASS_LIST.")
	if v, exist := os.LookupEnv("PASS_LIST"); exist {
		slog.Debug("Found PASS_LIST in environment.", "origin", v)
		PASS_LIST = parseList(v)
	}
	slog.Debug("PASS_LIST initialized.", "PASS_LIST", PASS_LIST)
	slog.Debug("Loading JSDELIVR_MIRROR.")
	if v, exist := os.LookupEnv("JSDELIVR_MIRROR"); exist {
		slog.Debug("Found JSDELIVR_MIRROR in environment.", "origin", v)
		USE_JSDELIVR_AS_MIRROR_FOR_BRANCHES = v == "1" || v == "true" || v == "True" || v == "TRUE"
	}
	slog.Debug("JSDELIVR_MIRROR initialized.", "JSDELIVR_MIRROR", USE_JSDELIVR_AS_MIRROR_FOR_BRANCHES)
	slog.Debug("Loading SIZE_LIMIT.")
	if v, exist := os.LookupEnv("SIZE_LIMIT"); exist {
		slog.Debug("Found SIZE_LIMIT in environment.", "origin", v)
		n, err := strconv.Atoi(v)
		if err != nil {
			slog.Debug("SIZE_LIMIT is not an integer. Trying to convert to xxG/xxM")
			c := int64(1)
			ns := v
			if strings.HasSuffix(v, "G") {
				slog.Debug("SIZE_LIMIT ends with 'G'")
				c = 1024 * 1024 * 1024
				ns = v[0 : len(v)-1]
			} else if strings.HasSuffix(v, "M") {
				slog.Debug("SIZE_LIMIT ends with 'M'")
				c = 1024 * 1024
				ns = v[0 : len(v)-1]
			} else if strings.HasSuffix(v, "K") {
				slog.Debug("SIZE_LIMIT ends with 'K'")
				c = 1024
				ns = v[0 : len(v)-1]
			}
			n, err := strconv.Atoi(ns)
			if err != nil {
				slog.Error("Invalid environment variable: SIZE_LIMIT", "err", err)
				os.Exit(1)
			}
			SIZE_LIMIT = int64(n) * c
		} else {
			slog.Debug("SIZE_LIMIT is an integer.", "n", n)
			SIZE_LIMIT = int64(n)
		}
	}
	slog.Debug("SIZE_LIMIT initialized.", "SIZE_LIMIT", SIZE_LIMIT)
}

func init() {
	loadEnv()
}

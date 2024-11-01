package client

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/strutil"

	"github.com/essentialkaos/go-linenoise/v3"

	"github.com/essentialkaos/redy/v4"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config contains cli configuration
type Config struct {
	ID             int      // Instance ID
	Port           int      // Redis port
	DB             int      // DB index
	User           string   // ACL password
	Password       string   // ACL password
	Command        []string // Command
	HistoryFile    string   // Path to history file
	Timeout        int      // Connection timeout
	DisableMonitor bool     // Disable MONITOR command flag
	RawOutput      bool     // Raw output flag
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Max ops per second for monitor usage
const MONITOR_MAX_OPS uint64 = 1000

// ////////////////////////////////////////////////////////////////////////////////// //

// Prompt is prompt symbol
var Prompt = "> "

// UseColoredPrompt enable colors in CLI prompt
var UseColoredPrompt = false

// ////////////////////////////////////////////////////////////////////////////////// //

var unsupportedCommands = map[string]bool{
	"PSUBSCRIBE":   true,
	"PUBLISH":      true,
	"SUBSCRIBE":    true,
	"PUBSUB":       true,
	"PUNSUBSCRIBE": true,
	"UNSUBSCRIBE":  true,
}

var client *redy.Client

// ////////////////////////////////////////////////////////////////////////////////// //

// ExecRedisCmd simply execute given command
func ExecRedisCmd(cfg *Config) error {
	if len(cfg.Command) == 0 {
		return errors.New("Not enough command arguments")
	}

	cmd := cfg.Command[0]

	if unsupportedCommands[cmd] {
		return fmt.Errorf("RDS currently doesn't have native support of %s command", cmd)
	}

	if strings.ToUpper(cmd) == "MONITOR" {
		if cfg.DisableMonitor {
			return fmt.Errorf("Traffic on instance is too high (> %d op/s) for using monitor command", MONITOR_MAX_OPS)
		}

		return execMonitor(cfg, cmd)
	}

	return execCommand(cfg)
}

// RunRedisCli run interactive cli
func RunRedisCli(cfg *Config) error {
	prompt := getPrompt(cfg.ID, cfg.Port, cfg.DB)
	client := getClient(cfg.Port, time.Second*time.Duration(cfg.Timeout))

	err := client.Connect()

	if err != nil {
		return err
	}

	defer client.Close()

	configureClient(client, cfg)
	updateCommandsSupport(client)
	initCLIFeatures(cfg)

	var resp *redy.Resp

	for {
		input, err := linenoise.Line(prompt)

		if err != nil {
			break
		}

		if input == "" {
			continue
		}

		linenoise.AddHistory(input)

		cmd := strutil.Fields(input)

		if unsupportedCommands[cmd[0]] {
			fmt.Printf("\nRDS currently doesn't have native support of %s command\n\n", cmd[0])
			continue
		}

		if strings.ToUpper(cmd[0]) == "MONITOR" {
			if cfg.DisableMonitor {
				fmt.Printf("\nTraffic on instance is too high (> %d op/s) for using monitor command\n\n", MONITOR_MAX_OPS)
			} else {
				fmt.Println("")
				execMonitor(cfg, cmd[0])
			}

			continue
		}

		switch len(cmd) {
		case 1:
			resp = client.Cmd(cmd[0])
		default:
			resp = client.Cmd(cmd[0], convertCommandSlice(cmd[1:]))
		}

		fmt.Print("\n" + formatResp(resp, false) + "\n")

		if cmd[0] == "SELECT" && !resp.HasType(redy.ERR) {
			// Ignore error because redis return ok response
			db, _ := strconv.Atoi(cmd[1])
			prompt = getPrompt(cfg.ID, cfg.Port, db)
		}
	}

	if cfg.HistoryFile != "" {
		linenoise.SaveHistory(cfg.HistoryFile)
	}

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// execCommand execs one command
func execCommand(cfg *Config) error {
	client := getClient(cfg.Port, time.Second*time.Duration(cfg.Timeout))

	err := client.Connect()

	if err != nil {
		return err
	}

	defer client.Close()

	configureClient(client, cfg)

	var resp *redy.Resp

	switch len(cfg.Command) {
	case 1:
		resp = client.Cmd(cfg.Command[0])
	default:
		resp = client.Cmd(cfg.Command[0], convertCommandSlice(cfg.Command[1:]))
	}

	if resp.Err != nil {
		return resp.Err
	}

	fmt.Print(formatResp(resp, cfg.RawOutput))

	return nil
}

// getClient return Redy client
func getClient(port int, timeout time.Duration) *redy.Client {
	if client == nil {
		client = &redy.Client{}
	}

	client.Addr = "127.0.0.1:" + strconv.Itoa(port)

	if timeout > 0 {
		client.WriteTimeout = timeout
		client.ReadTimeout = timeout
	} else {
		client.WriteTimeout = time.Minute
		client.ReadTimeout = time.Minute
	}

	return client
}

// configureClient configure client
func configureClient(client *redy.Client, cfg *Config) error {
	var resp *redy.Resp

	if cfg.User != "" && cfg.Password != "" {
		resp = client.Cmd("AUTH", cfg.User, cfg.Password)

		if resp.Err != nil {
			return resp.Err
		}
	}

	if cfg.DB != 0 {
		resp = client.Cmd("SELECT", cfg.DB)

		if resp.Err != nil {
			return resp.Err
		}
	}

	return nil
}

// execMonitor exec monitor command (connection not be closed)
func execMonitor(cfg *Config, cmd string) error {
	conn, err := net.DialTimeout(
		"tcp", "127.0.0.1:"+strconv.Itoa(cfg.Port),
		time.Second*time.Duration(cfg.Timeout),
	)

	if err != nil {
		return err
	}

	defer conn.Close()

	if cfg.Password != "" {
		conn.Write([]byte("AUTH " + cfg.User + " " + cfg.Password + "\n"))
	}

	conn.Write([]byte(cmd + "\n"))
	connbuf := bufio.NewReader(conn)

	for {
		str, err := connbuf.ReadString('\n')

		if len(str) > 0 {
			if str == "+OK\r\n" {
				continue
			}

			fmt.Printf("%s", str[1:])
		}

		if err != nil {
			break
		}
	}

	return nil
}

// getPrompt return string with prompt
func getPrompt(id, port, db int) string {
	switch {
	case UseColoredPrompt && db != 0:
		return fmtc.Sprintf("{s*}%d{!*}{s-}:%d{s}[%d]{!}"+Prompt, id, port, db)
	case UseColoredPrompt && db == 0:
		return fmtc.Sprintf("{s*}%d{!*}{s-}:%d{!}"+Prompt, id, port)
	case !UseColoredPrompt && db != 0:
		return fmt.Sprintf("%d:%d[%d]"+Prompt, id, port, db)
	default:
		return fmt.Sprintf("%d:%d"+Prompt, id, port)
	}
}

// formatResp format redis response and return response as string
func formatResp(r *redy.Resp, raw bool) string {
	switch {
	case r.HasType(redy.ARRAY):
		return formatArrayResp(r, 0, raw)
	case r.HasType(redy.STR):
		return formatStrResp(r, raw)
	case r.HasType(redy.INT):
		return formatIntResp(r, raw)
	case r.HasType(redy.ERR):
		return formatErrorResp(r, raw)
	case r.HasType(redy.NIL):
		return formatNilResp(r, raw)
	default:
		return formatStrResp(r, true)
	}
}

// formatStrResp format str/bulk str response
func formatStrResp(r *redy.Resp, raw bool) string {
	str, _ := r.Str()
	str = strings.ReplaceAll(str, "%", "%%")

	if raw {
		return str + "\n"
	}

	return fmtc.Sprintf("{y}\"%s\"{!}\n", str)
}

// formatInt format integer response
func formatIntResp(r *redy.Resp, raw bool) string {
	i, _ := r.Int64()
	return fmtc.Sprintf("{c}%d{!}\n", i)
}

// formatErrorResp format error response
func formatErrorResp(r *redy.Resp, raw bool) string {
	if raw {
		return r.Err.Error() + "\n"
	}

	return fmtc.Sprintf("{r}%s{!}\n", r.Err.Error())
}

// formatNilResp format nil response
func formatNilResp(r *redy.Resp, raw bool) string {
	if raw {
		return "(nil)\n"
	}

	return fmtc.Sprintf("{m}Nil{!}\n")
}

// formatArrayResp format array response
func formatArrayResp(r *redy.Resp, prefixSize int, raw bool) string {
	items, err := r.Array()

	if err != nil || len(items) == 0 {
		return fmtc.Sprintf("{s}(empty list or set){!}\n")
	}

	var result string

	if raw {
		for _, item := range items {
			switch {
			case item.HasType(redy.STR_BULK):
				result += formatArrayResp(item, 0, raw)
			case item.HasType(redy.INT):
				v, _ := item.Int()
				result += strconv.Itoa(v) + "\n"
			default:
				v, _ := item.Str()
				result += v + "\n"
			}
		}

		return result
	}

	prefix := strings.Repeat(" ", prefixSize)
	numSize := fmtutil.CountDigits(len(items))
	numFormat := fmt.Sprintf("{s-}%%%dd){!} ", numSize)

	for index, item := range items {
		if prefixSize == 0 || index != 0 {
			result += prefix
		}

		switch {
		case item.HasType(redy.ARRAY):
			result += fmtc.Sprintf(numFormat, index+1) + formatArrayResp(item, prefixSize+numSize+2, false)
		case item.HasType(redy.STR):
			v, _ := item.Str()
			result += fmtc.Sprintf(numFormat+"{y}\"%s\"{!}\n", index+1, v)
		case item.HasType(redy.INT):
			v, _ := item.Int()
			result += fmtc.Sprintf(numFormat+"{c}%d{!}\n", index+1, v)
		default:
			v, _ := item.Str()
			result += fmtc.Sprintf(numFormat+"%s\n", index+1, v)
		}
	}

	return result
}

// initCLIFeatures add autocompele and hints for user input
func initCLIFeatures(cfg *Config) {
	linenoise.SetCompletionHandler(autocompleteHandler)
	linenoise.SetHintHandler(hintHandler)

	if cfg.HistoryFile != "" && fsutil.CheckPerms("FRS", cfg.HistoryFile) {
		linenoise.LoadHistory(cfg.HistoryFile)
	}
}

// autocompleteHandler is autocomplete handler function
func autocompleteHandler(input string) []string {
	if strings.TrimSpace(input) == "" {
		return getAvailableCommands()
	}

	return getSuggestions(input)
}

// hintHandler is hints handler function
func hintHandler(input string) string {
	if input == "" {
		return ""
	}

	for _, command := range getCommands() {
		if !strings.HasPrefix(strings.ToUpper(input), command.Name) {
			continue
		}

		if len(command.Params) == 0 || strings.ContainsAny(input, "\"'") {
			continue
		}

		fullCommandSlice := append([]string{command.Name}, command.Params...)
		inputSlice := strutil.Fields(input)
		startFrom := len(inputSlice) - strings.Count(command.Name, " ")

		for i := 0; i < startFrom; i++ {
			if i == len(command.Params) {
				break
			}

			if strings.Contains(command.Params[i], "...") {
				startFrom = i + 1
				break
			}
		}

		if startFrom > len(fullCommandSlice) {
			return ""
		}

		if strutil.Tail(input, 1) == " " {
			return strings.Join(fullCommandSlice[startFrom:], " ")
		}

		return " " + strings.Join(fullCommandSlice[startFrom:], " ")
	}

	return ""
}

// getSuggestions return slice with command suggestions
func getSuggestions(input string) []string {
	var result []string

	for _, command := range getCommands() {
		if strings.HasPrefix(command.Name, strings.ToUpper(input)) {
			result = append(result, command.Name)
		}
	}

	return result
}

// convertCommandSlice convert command string slice to slice with any
func convertCommandSlice(cmd []string) []any {
	var result []any

	for _, c := range cmd {
		ci, err := strconv.Atoi(c)

		if err == nil {
			result = append(result, ci)
			continue
		}

		cf, err := strconv.ParseFloat(c, 64)

		if err == nil {
			result = append(result, cf)
			continue
		}

		result = append(result, c)
	}

	return result
}

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/artifacts"
	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "init":
		runInit(os.Args[2:])
	case "append":
		runAppend(os.Args[2:])
	case "query":
		runQuery(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "artifact":
		runArtifact(os.Args[2:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "stateledger <command> [options]")
	fmt.Fprintln(os.Stderr, "commands: init, append, query, verify, artifact")
}

func defaultDBPath() string {
	return filepath.Join("data", "ledger.db")
}

func defaultArtifactsPath() string {
	return "artifacts"
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	artifactsPath := fs.String("artifacts", defaultArtifactsPath(), "path to artifacts store")
	_ = fs.Parse(args)

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*artifactsPath, 0o755); err != nil {
		fatal(err)
	}

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	if err := l.InitSchema(); err != nil {
		fatal(err)
	}

	fmt.Println("initialized")
}

func runAppend(args []string) {
	fs := flag.NewFlagSet("append", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	rtype := fs.String("type", "", "record type")
	source := fs.String("source", "", "record source")
	payloadFile := fs.String("payload-file", "", "path to payload file")
	payloadJSON := fs.String("payload-json", "", "payload JSON string")
	timestamp := fs.Int64("time", 0, "unix timestamp (seconds)")
	_ = fs.Parse(args)

	if *rtype == "" {
		fatal(errors.New("--type is required"))
	}

	payload, err := readPayload(*payloadFile, *payloadJSON)
	if err != nil {
		fatal(err)
	}

	ts := *timestamp
	if ts == 0 {
		ts = time.Now().Unix()
	}

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	rec, err := l.Append(ledger.RecordInput{
		Timestamp: ts,
		Type:      *rtype,
		Source:    *source,
		Payload:   payload,
	})
	if err != nil {
		fatal(err)
	}

	out, _ := json.Marshal(rec)
	fmt.Println(string(out))
}

func runQuery(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	id := fs.Int64("id", 0, "record id")
	since := fs.Int64("since", 0, "unix timestamp (seconds)")
	until := fs.Int64("until", 0, "unix timestamp (seconds)")
	limit := fs.Int("limit", 100, "max records")
	_ = fs.Parse(args)

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	if *id > 0 {
		rec, err := l.GetByID(*id)
		if err != nil {
			fatal(err)
		}
		out, _ := json.Marshal(rec)
		fmt.Println(string(out))
		return
	}

	recs, err := l.List(ledger.ListQuery{
		Since: *since,
		Until: *until,
		Limit: *limit,
	})
	if err != nil {
		fatal(err)
	}

	enc := json.NewEncoder(os.Stdout)
	for _, rec := range recs {
		_ = enc.Encode(rec)
	}
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	_ = fs.Parse(args)

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	result, err := l.VerifyChain()
	if err != nil {
		fatal(err)
	}

	out, _ := json.Marshal(result)
	fmt.Println(string(out))
}

func runArtifact(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "artifact subcommands: put")
		os.Exit(2)
	}

	switch args[0] {
	case "put":
		runArtifactPut(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "unknown artifact command")
		os.Exit(2)
	}
}

func runArtifactPut(args []string) {
	fs := flag.NewFlagSet("artifact put", flag.ExitOnError)
	artifactsPath := fs.String("artifacts", defaultArtifactsPath(), "path to artifacts store")
	source := fs.String("file", "", "file to store")
	_ = fs.Parse(args)

	if *source == "" {
		fatal(errors.New("--file is required"))
	}

	if err := os.MkdirAll(*artifactsPath, 0o755); err != nil {
		fatal(err)
	}

	info, err := artifacts.Store(*artifactsPath, *source)
	if err != nil {
		fatal(err)
	}

	out, _ := json.Marshal(info)
	fmt.Println(string(out))
}

func readPayload(filePath, inline string) (string, error) {
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if inline != "" {
		return inline, nil
	}
	return "", errors.New("payload required: --payload-file or --payload-json")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

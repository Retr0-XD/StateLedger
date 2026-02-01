package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/api"
	"github.com/Retr0-XD/StateLedger/internal/artifacts"
	"github.com/Retr0-XD/StateLedger/internal/collectors"
	"github.com/Retr0-XD/StateLedger/internal/ledger"
	"github.com/Retr0-XD/StateLedger/internal/manifest"
	"github.com/Retr0-XD/StateLedger/internal/sources"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "init":
		runInit(os.Args[2:])
	case "collect":
		runCollect(os.Args[2:])
	case "capture":
		runCapture(os.Args[2:])
	case "manifest":
		runManifest(os.Args[2:])
	case "append":
		runAppend(os.Args[2:])
	case "query":
		runQuery(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "snapshot":
		runSnapshot(os.Args[2:])
	case "advisory":
		runAdvisory(os.Args[2:])
	case "audit":
		runAudit(os.Args[2:])
	case "artifact":
		runArtifact(os.Args[2:])
	case "server":
		runServer(os.Args[2:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "stateledger <command> [options]")
	fmt.Fprintln(os.Stderr, "commands: init, collect, capture, manifest, append, query, verify, snapshot, advisory, audit, artifact, server")
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

func runCollect(args []string) {
	fs := flag.NewFlagSet("collect", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	kind := fs.String("kind", "", "collector kind: code|config|environment|mutation")
	source := fs.String("source", "", "record source")
	payloadFile := fs.String("payload-file", "", "path to payload file (JSON)")
	payloadJSON := fs.String("payload-json", "", "payload JSON string")
	timestamp := fs.Int64("time", 0, "unix timestamp (seconds)")
	_ = fs.Parse(args)

	if *kind == "" {
		fatal(errors.New("--kind is required"))
	}

	raw, err := readPayload(*payloadFile, *payloadJSON)
	if err != nil {
		fatal(err)
	}

	var recordType string
	var payload string
	if recordType, payload, err = buildCollectorPayload(*kind, raw); err != nil {
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
		Type:      recordType,
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

func buildCollectorPayload(kind, raw string) (string, string, error) {
	switch kind {
	case "code":
		var payload collectors.CodePayload
		if err := collectors.ParseJSON(raw, &payload); err != nil {
			return "", "", err
		}
		if err := payload.Validate(); err != nil {
			return "", "", err
		}
		body, err := collectors.MarshalPayload(payload)
		return "code", body, err
	case "config":
		var payload collectors.ConfigPayload
		if err := collectors.ParseJSON(raw, &payload); err != nil {
			return "", "", err
		}
		if err := payload.Validate(); err != nil {
			return "", "", err
		}
		body, err := collectors.MarshalPayload(payload)
		return "config", body, err
	case "environment":
		var payload collectors.EnvironmentPayload
		if err := collectors.ParseJSON(raw, &payload); err != nil {
			return "", "", err
		}
		if err := payload.Validate(); err != nil {
			return "", "", err
		}
		body, err := collectors.MarshalPayload(payload)
		return "environment", body, err
	case "mutation":
		var payload collectors.MutationPayload
		if err := collectors.ParseJSON(raw, &payload); err != nil {
			return "", "", err
		}
		if err := payload.Validate(); err != nil {
			return "", "", err
		}
		body, err := collectors.MarshalPayload(payload)
		return "mutation", body, err
	default:
		return "", "", fmt.Errorf("unknown kind: %s", kind)
	}
}

func runCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	kind := fs.String("kind", "", "capture kind: code|config|environment")
	path := fs.String("path", "", "path to capture from (repo, config file, etc)")
	_ = fs.Parse(args)

	if *kind == "" {
		fatal(errors.New("--kind is required"))
	}
	if *path == "" {
		fatal(errors.New("--path is required"))
	}

	result, err := sources.CaptureFromManifest(*kind, *path, nil)
	if err != nil {
		fatal(err)
	}

	out, _ := json.Marshal(result)
	fmt.Println(string(out))
}

func runManifest(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "manifest subcommands: create, run, show")
		os.Exit(2)
	}

	switch args[0] {
	case "create":
		runManifestCreate(args[1:])
	case "run":
		runManifestRun(args[1:])
	case "show":
		runManifestShow(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "unknown manifest command")
		os.Exit(2)
	}
}

func runManifestCreate(args []string) {
	fs := flag.NewFlagSet("manifest create", flag.ExitOnError)
	name := fs.String("name", "default", "manifest name")
	output := fs.String("output", "manifest.json", "output file")
	_ = fs.Parse(args)

	m := manifest.NewManifest(*name)
	m.AddCollector("code", ".", nil)
	m.AddCollector("environment", "", nil)

	json, err := m.ToJSON()
	if err != nil {
		fatal(err)
	}

	if err := os.WriteFile(*output, []byte(json), 0o644); err != nil {
		fatal(err)
	}

	fmt.Println("created: " + *output)
}

func runManifestRun(args []string) {
	fs := flag.NewFlagSet("manifest run", flag.ExitOnError)
	manifestPath := fs.String("file", "manifest.json", "manifest file")
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	source := fs.String("source", "manifest-run", "record source identifier")
	_ = fs.Parse(args)

	m, err := manifest.LoadManifest(*manifestPath)
	if err != nil {
		fatal(err)
	}

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	for _, c := range m.Collectors {
		result, _ := sources.CaptureFromManifest(c.Kind, c.Source, c.Params)

		if result.Error != "" {
			fmt.Fprintf(os.Stderr, "error capturing %s from %s: %s\n", c.Kind, c.Source, result.Error)
			continue
		}

		rec, err := l.Append(ledger.RecordInput{
			Timestamp: time.Now().Unix(),
			Type:      c.Kind,
			Source:    *source,
			Payload:   result.Payload,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error appending %s: %s\n", c.Kind, err)
			continue
		}

		out, _ := json.Marshal(rec)
		fmt.Println(string(out))
	}
}

func runManifestShow(args []string) {
	fs := flag.NewFlagSet("manifest show", flag.ExitOnError)
	manifestPath := fs.String("file", "manifest.json", "manifest file")
	_ = fs.Parse(args)

	m, err := manifest.LoadManifest(*manifestPath)
	if err != nil {
		fatal(err)
	}

	out, err := m.ToJSON()
	if err != nil {
		fatal(err)
	}

	fmt.Println(out)
}
func runSnapshot(args []string) {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	targetTime := fs.Int64("time", 0, "unix timestamp (seconds, 0=now)")
	_ = fs.Parse(args)

	if *targetTime == 0 {
		*targetTime = time.Now().Unix()
	}

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	rec := ledger.New(l)
	report := rec.ReconstructAtTime(*targetTime)

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
}

func runAdvisory(args []string) {
	fs := flag.NewFlagSet("advisory", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	targetTime := fs.Int64("time", 0, "unix timestamp (seconds, 0=now)")
	_ = fs.Parse(args)

	if *targetTime == 0 {
		*targetTime = time.Now().Unix()
	}

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	rec := ledger.New(l)
	report := rec.ReconstructAtTime(*targetTime)

	// Analyze determinism
	envAnalysis := ledger.AnalyzeEnvironment(report.State.Environment)
	codeAnalysis := ledger.AnalyzeCode(report.State.Code)
	configAnalysis := ledger.AnalyzeConfig(report.State.Config)
	summary := ledger.SummarizeAnalyses(envAnalysis, codeAnalysis, configAnalysis)

	// Print analysis
	fmt.Println("=== Determinism Advisory ===")
	fmt.Println(ledger.ReportJSON(summary))
	fmt.Println("\n=== Explanation ===")
	fmt.Println(rec.ExplainFailure(report))
}

func runAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	targetTime := fs.Int64("time", 0, "unix timestamp (seconds, 0=now)")
	output := fs.String("out", "", "write bundle to file")
	_ = fs.Parse(args)

	if *targetTime == 0 {
		*targetTime = time.Now().Unix()
	}

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	rec := ledger.New(l)
	bundle, err := rec.ExportAuditBundle(*targetTime)
	if err != nil {
		fatal(err)
	}

	json, err := bundle.ToJSON()
	if err != nil {
		fatal(err)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(json), 0o644); err != nil {
			fatal(err)
		}
		fmt.Println("written: " + *output)
		return
	}

	fmt.Println(json)
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "path to ledger database")
	addr := fs.String("addr", ":8080", "server address (host:port)")
	_ = fs.Parse(args)

	l, err := ledger.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer l.Close()

	server := api.NewServer(l, *addr)
	if err := server.Start(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

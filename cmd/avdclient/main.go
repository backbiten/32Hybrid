// cmd/avdclient — 32Hybrid AVD client CLI.
//
// The AVD client runs inside the AVD session and submits EXE run requests to
// the control plane over gRPC.
//
// Commands:
//
//	avdclient submit --exe <path> [--name <exe_name>] [--timeout <sec>] [-- arg1 arg2 ...]
//	avdclient get    --run-id <id>
//	avdclient list
//
// Configuration:
//
//	avdclient --config <path> <command>
//	avdclient --addr <host:port> <command>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cpv1 "github.com/backbiten/32Hybrid/gen/controlplane/v1"
)

func main() {
	// Top-level flags
	addr := flag.String("addr", "localhost:50051", "Control plane gRPC address (host:port)")
	timeout := flag.Duration("timeout", 30*time.Second, "gRPC call timeout")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()
	client := cpv1.NewControlPlaneServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	switch args[0] {
	case "submit":
		runSubmit(ctx, client, args[1:])
	case "get":
		runGet(ctx, client, args[1:])
	case "list":
		runList(ctx, client, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage()
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// submit
// ─────────────────────────────────────────────────────────────────────────────

func runSubmit(ctx context.Context, client cpv1.ControlPlaneServiceClient, args []string) {
	fs := flag.NewFlagSet("submit", flag.ExitOnError)
	exePath := fs.String("exe", "", "Path to the EXE to submit (required)")
	exeName := fs.String("name", "", "Override EXE name stored in blob (default: basename of --exe)")
	sha256Override := fs.String("sha256", "", "Expected SHA-256 of the EXE (optional verification)")
	timeoutSec := fs.Int("timeout", 300, "Process timeout in seconds")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: avdclient submit --exe <path> [--name <name>] [--timeout <sec>] [-- arg1 arg2 ...]")
		fs.PrintDefaults()
	}

	// Split on "--" to separate flags from positional args.
	splitIdx := -1
	for i, a := range args {
		if a == "--" {
			splitIdx = i
			break
		}
	}
	flagArgs := args
	var exeArgs []string
	if splitIdx >= 0 {
		flagArgs = args[:splitIdx]
		exeArgs = args[splitIdx+1:]
	}
	if err := fs.Parse(flagArgs); err != nil {
		log.Fatal(err)
	}

	if *exePath == "" {
		fmt.Fprintln(os.Stderr, "submit: --exe is required")
		fs.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*exePath)
	if err != nil {
		log.Fatalf("read exe %q: %v", *exePath, err)
	}

	name := *exeName
	if name == "" {
		name = lastPathSegment(*exePath)
	}

	resp, err := client.SubmitRun(ctx, &cpv1.SubmitRunRequest{
		ExeBytes:       data,
		ExeSha256:      *sha256Override,
		ExeName:        name,
		Args:           exeArgs,
		TimeoutSeconds: int32(*timeoutSec),
	})
	if err != nil {
		log.Fatalf("SubmitRun: %v", err)
	}
	fmt.Printf("run_id: %s\n", resp.RunId)
}

// ─────────────────────────────────────────────────────────────────────────────
// get
// ─────────────────────────────────────────────────────────────────────────────

func runGet(ctx context.Context, client cpv1.ControlPlaneServiceClient, args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	runID := fs.String("run-id", "", "Run ID to query (required)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: avdclient get --run-id <id>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *runID == "" {
		fmt.Fprintln(os.Stderr, "get: --run-id is required")
		fs.Usage()
		os.Exit(1)
	}

	resp, err := client.GetRun(ctx, &cpv1.GetRunRequest{RunId: *runID})
	if err != nil {
		log.Fatalf("GetRun: %v", err)
	}
	printRun(resp)
}

// ─────────────────────────────────────────────────────────────────────────────
// list
// ─────────────────────────────────────────────────────────────────────────────

func runList(ctx context.Context, client cpv1.ControlPlaneServiceClient, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	pageSize := fs.Int("n", 50, "Max number of runs to return")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	resp, err := client.ListRuns(ctx, &cpv1.ListRunsRequest{PageSize: int32(*pageSize)})
	if err != nil {
		log.Fatalf("ListRuns: %v", err)
	}
	if len(resp.Runs) == 0 {
		fmt.Println("no runs found")
		return
	}
	for _, r := range resp.Runs {
		printRun(r)
		fmt.Println("---")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func printRun(r *cpv1.GetRunResponse) {
	fmt.Printf("run_id:      %s\n", r.RunId)
	fmt.Printf("state:       %s\n", r.State)
	fmt.Printf("submitted:   %s\n", formatUnix(r.SubmittedAt))
	fmt.Printf("started:     %s\n", formatUnix(r.StartedAt))
	fmt.Printf("finished:    %s\n", formatUnix(r.FinishedAt))
	fmt.Printf("exit_code:   %d\n", r.ExitCode)
	if r.ErrorMessage != "" {
		fmt.Printf("error:       %s\n", r.ErrorMessage)
	}
	if r.ExitJsonBlobPath != "" {
		fmt.Printf("exit_json:   %s\n", r.ExitJsonBlobPath)
	}
	if r.StdoutBlobPath != "" {
		fmt.Printf("stdout:      %s\n", r.StdoutBlobPath)
	}
	if r.StderrBlobPath != "" {
		fmt.Printf("stderr:      %s\n", r.StderrBlobPath)
	}
}

func formatUnix(ts int64) string {
	if ts == 0 {
		return "(not yet)"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func lastPathSegment(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func usage() {
	fmt.Fprintln(os.Stderr, `32Hybrid AVD client

Usage:
  avdclient [--addr <host:port>] <command> [flags]

Commands:
  submit  Submit an EXE run
  get     Get run status
  list    List recent runs

Flags:`)
	flag.PrintDefaults()
}

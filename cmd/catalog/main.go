// Command catalog is a staff CLI tool to interact with the supplier catalog.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"supplier-catalog/internal/catalog"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "API server base URL")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cli := &CLI{
		baseURL: strings.TrimRight(*serverURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}

	cmd := args[0]
	cmdArgs := args[1:]

	var err error
	switch cmd {
	case "start":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: catalog start <update-id>")
			os.Exit(1)
		}
		err = cli.Start(cmdArgs[0])
	case "resume":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: catalog resume <run-id>")
			os.Exit(1)
		}
		id, parseErr := strconv.ParseInt(cmdArgs[0], 10, 64)
		if parseErr != nil {
			fmt.Printf("Invalid run ID: %v\n", cmdArgs[0])
			os.Exit(1)
		}
		err = cli.Resume(id)
	case "abandon":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: catalog abandon <run-id>")
			os.Exit(1)
		}
		id, parseErr := strconv.ParseInt(cmdArgs[0], 10, 64)
		if parseErr != nil {
			fmt.Printf("Invalid run ID: %v\n", cmdArgs[0])
			os.Exit(1)
		}
		err = cli.Abandon(id)
	case "runs":
		err = cli.ListRuns()
	case "changes":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: catalog changes <run-id>")
			os.Exit(1)
		}
		id, parseErr := strconv.ParseInt(cmdArgs[0], 10, 64)
		if parseErr != nil {
			fmt.Printf("Invalid run ID: %v\n", cmdArgs[0])
			os.Exit(1)
		}
		err = cli.ListChanges(id)
	case "problems":
		status := ""
		if len(cmdArgs) > 0 {
			status = cmdArgs[0]
		}
		err = cli.ListProblems(status)
	case "resolve":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: catalog resolve <problem-id>")
			os.Exit(1)
		}
		id, parseErr := strconv.ParseInt(cmdArgs[0], 10, 64)
		if parseErr != nil {
			fmt.Printf("Invalid problem ID: %v\n", cmdArgs[0])
			os.Exit(1)
		}
		err = cli.Resolve(id)
	case "products":
		status := ""
		if len(cmdArgs) > 0 {
			status = cmdArgs[0]
		}
		err = cli.ListProducts(status)
	default:
		fmt.Printf("Unknown command: %q\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`Usage: catalog [-server http://localhost:8080] <command> [arguments]

Commands:
  start <update-id>     Start importing a supplier update
  resume <run-id>       Resume an interrupted update from its last batch
  abandon <run-id>      Abandon an interrupted update
  runs                  List all update runs and their status
  changes <run-id>      Show what changed in a specific run
  problems [open|all]   List problems needing attention
  resolve <problem-id>  Mark a problem as resolved
  products [active|withdrawn] List products in local catalog`)
}

type CLI struct {
	baseURL string
	client  *http.Client
}

func (c *CLI) Start(updateID string) error {
	var run catalog.Run
	err := c.post("/api/runs", map[string]string{"update_id": updateID}, &run)
	if err != nil {
		return err
	}
	fmt.Printf("Update #%d started for %s\n", run.ID, run.SupplierUpdateID)
	return nil
}

func (c *CLI) Resume(runID int64) error {
	var run catalog.Run
	err := c.post(fmt.Sprintf("/api/runs/%d/resume", runID), nil, &run)
	if err != nil {
		return err
	}
	fmt.Printf("Update #%d resumed from batch %d\n", run.ID, run.NextBatch)
	return nil
}

func (c *CLI) Abandon(runID int64) error {
	var run catalog.Run
	err := c.post(fmt.Sprintf("/api/runs/%d/abandon", runID), nil, &run)
	if err != nil {
		return err
	}
	fmt.Printf("Update #%d abandoned\n", run.ID)
	return nil
}

func (c *CLI) ListRuns() error {
	var runs []catalog.Run
	if err := c.get("/api/runs", &runs); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUPDATE_ID\tSTATUS\tBATCHES\tCREATED\tUPDATED\tWITHDRAWN\tSKIPPED\tPROBLEMS")
	for _, r := range runs {
		done := r.NextBatch - 1
		total := strconv.Itoa(r.TotalBatches)
		if r.TotalBatches == 0 {
			total = "?"
		}
		batches := fmt.Sprintf("%d/%s", done, total)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			r.ID, r.SupplierUpdateID, r.Status, batches,
			r.Counts.Created, r.Counts.Updated, r.Counts.Withdrawn, r.Counts.Skipped, r.OpenProblems)
	}
	return w.Flush()
}

func (c *CLI) ListChanges(runID int64) error {
	var changes []catalog.Change
	if err := c.get(fmt.Sprintf("/api/runs/%d/changes", runID), &changes); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BATCH\tSKU\tACTION\tDETAIL")
	for _, ch := range changes {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", ch.Batch, ch.SKU, ch.Action, ch.Detail)
	}
	return w.Flush()
}

func (c *CLI) ListProblems(status string) error {
	path := "/api/problems"
	if status != "" && status != "all" {
		path += "?status=" + status
	}
	var problems []catalog.Problem
	if err := c.get(path, &problems); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tRUN\tBATCH\tSKU\tKIND\tSTATUS\tMESSAGE")
	for _, p := range problems {
		sku := p.SKU
		if sku == "" {
			sku = "-"
		}
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t%s\t%s\n", p.ID, p.RunID, p.Batch, sku, p.Kind, p.Status, p.Message)
	}
	return w.Flush()
}

func (c *CLI) Resolve(id int64) error {
	var res map[string]any
	err := c.post(fmt.Sprintf("/api/problems/%d/resolve", id), nil, &res)
	if err != nil {
		return err
	}
	fmt.Printf("Problem #%d marked as resolved\n", id)
	return nil
}

func (c *CLI) ListProducts(status string) error {
	path := "/api/products"
	if status != "" && status != "all" {
		path += "?status=" + status
	}
	var products []catalog.Product
	if err := c.get(path, &products); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SKU\tNAME\tPRICE\tSTATUS\tSUPPLIER_UPDATED_AT")
	for _, p := range products {
		price := fmt.Sprintf("%.2f %s", float64(p.PriceCents)/100, p.Currency)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.SKU, p.Name, price, p.Status, p.SupplierUpdatedAt.Format(time.RFC3339))
	}
	return w.Flush()
}

func (c *CLI) get(path string, out any) error {
	res, err := c.client.Get(c.baseURL + path)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return decodeResponse(res, out)
}

func (c *CLI) post(path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return decodeResponse(res, out)
}

func decodeResponse(res *http.Response, out any) error {
	if res.StatusCode >= 400 {
		var errData struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(res.Body).Decode(&errData); err == nil && errData.Error != "" {
			return fmt.Errorf("HTTP %d: %s", res.StatusCode, errData.Error)
		}
		return fmt.Errorf("HTTP %d", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

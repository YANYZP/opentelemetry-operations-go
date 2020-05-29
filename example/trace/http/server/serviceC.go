package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opencensus.io/plugin/ochttp"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
	projectID := os.Getenv("PROJECT_ID")

	// Create Google Cloud Trace exporter to be able to retrieve
	// the collected spans.
	exporter, err := cloudtrace.NewExporter(
		cloudtrace.WithProjectID(projectID),
	)
	if err != nil {
		log.Fatal(err)
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

func readFile(fileName string) map[string]map[string]string {

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var itemPriceMap = make(map[string]map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		priceInfo := scanner.Text()
		words := strings.Fields(priceInfo)
		fmt.Println(priceInfo)
		if len(words) != 3 {
			fmt.Println("Wrong format")
			continue
		}
		if itemPriceMap[words[1]] == nil {
			itemPriceMap[words[1]] = make(map[string]string)
		}
		itemPriceMap[words[1]][words[0]] = words[2]
	}

	// for k, v := range itemPriceMap {
	// 	for _, vv := range v {
	// 		fmt.Println(k, v, vv)
	// 	}
	// }

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return itemPriceMap
}

func main() {
	initTracer()

	itemPriceMap := readFile("price.txt")

	tr := global.TraceProvider().Tracer("cloudtrace/example/server")

	urlHandler := func(w http.ResponseWriter, req *http.Request) {
		attrs, entries, spanCtx := httptrace.Extract(req.Context(), req)

		fmt.Println("service C url", req.URL.Path)

		itemVendorStr := req.URL.Path[1:] // item#vendor1#vendor2...
		fmt.Println("service c: itemVendorStr = " + itemVendorStr)

		req = req.WithContext(correlation.ContextWithMap(req.Context(), correlation.NewMap(correlation.MapUpdate{
			MultiKV: entries,
		})))

		ctx, span := tr.Start(
			trace.ContextWithRemoteSpanContext(req.Context(), spanCtx),
			"hello",
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		span.AddEvent(ctx, "handling this...")

		infoArray := strings.Split(itemVendorStr, "#")

		if len(infoArray) < 2 {
			_, _ = io.WriteString(w, "Service C fails to find enough info\n")
			return
		}
		itemName := infoArray[0]

		vendorPriceMap, ok := itemPriceMap[itemName]

		if !ok {
			_, _ = io.WriteString(w, "Service C: Not finding vendors for this item\n")
			return
		}

		vendorPriceStrBuilder := strings.Builder{}

		for i := 1; i < len(infoArray); i++ {
			vendorName := infoArray[i]
			price, okok := vendorPriceMap[vendorName]

			if !okok {
				fmt.Println("service c: fail to find price of" + itemName + " in " + vendorName)
			} else {
				vendorPriceStrBuilder.WriteString(price + "dollar at" + vendorName + "\n")
			}

		}
		_, _ = io.WriteString(w, vendorPriceStrBuilder.String())
	}

	http.HandleFunc("/", urlHandler)

	// Use an ochttp.Handler in order to instrument OpenCensus for incoming
	// requests.
	httpHandler := &ochttp.Handler{
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
	}
	if err := http.ListenAndServe(":7777", httpHandler); err != nil {
		panic(err)
	}
}

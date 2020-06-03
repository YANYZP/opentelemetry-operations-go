package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
	"google.golang.org/grpc/codes"
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

func readFromFile(fileName string) map[string][]string {
	file, err := os.Open("vendor.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var vendorOfItemMap = make(map[string][]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		itemVendorInfo := scanner.Text()
		words := strings.Fields(itemVendorInfo)
		fmt.Println(itemVendorInfo)
		for i := 1; i < len(words); i++ {
			fmt.Println("vendor", words[i])
			vendorOfItemMap[words[0]] = append(vendorOfItemMap[words[0]], words[i])
		}
		fmt.Println("vendor list", vendorOfItemMap[words[0]])
	}

	// for k, v := range vendorOfItemMap {
	// 	for _, vv := range v {
	// 		fmt.Println(k, vv)
	// 	}
	// }

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return vendorOfItemMap
}

func generateURL(ingredientName string) (string, bool) {
	vendorOfItemMap := readFromFile("vendor.txt")

	vendorNameList, ok := vendorOfItemMap[ingredientName]

	if !ok {
		return "Service B: No vendor info about " + ingredientName + "\n", false
	}

	URLStrBuilder := strings.Builder{}
	URLStrBuilder.WriteString(ingredientName)
	URLStrBuilder.WriteString("/")

	for _, vendorName := range vendorNameList {
		URLStrBuilder.WriteString(vendorName)
		URLStrBuilder.WriteString("/")
	}

	URLStr := URLStrBuilder.String()
	URLStr = URLStr[:len(URLStr)-1]

	fmt.Println("vendorNameList", vendorNameList)

	fmt.Println("sending url", URLStr, "to service c")

	return URLStr, true
}

func main() {
	initTracer()

	tr := global.TraceProvider().Tracer("cloudtrace/example/server")

	urlHandler := func(w http.ResponseWriter, req *http.Request) {
		attrs, entries, spanCtx := httptrace.Extract(req.Context(), req)

		ingredientName := req.URL.Path[1:]

		queryStr, isItemFound := generateURL(ingredientName)

		if !isItemFound {
			_, _ = io.WriteString(w, "Service B: No vendor info about "+ingredientName+"\n")
			return
		}

		req = req.WithContext(correlation.ContextWithMap(req.Context(), correlation.NewMap(correlation.MapUpdate{
			MultiKV: entries,
		})))

		ctx, span := tr.Start(
			trace.ContextWithRemoteSpanContext(req.Context(), spanCtx),
			"serviceB span",
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		span.AddEvent(ctx, "handling this...")

		tr := global.TraceProvider().Tracer("cloudtrace/example/client")

		client := http.DefaultClient
		// ctx := correlation.NewContext(context.Background(),
		// 	kv.String("username", "donuts"),
		// )

		var body []byte

		err := tr.WithSpan(ctx, "service B",
			func(ctx context.Context) error {
				// make sure the IP of service C
				req, _ := http.NewRequest("GET", "http://34.67.111.154:7777/"+queryStr, nil)

				ctx, req = httptrace.W3C(ctx, req)
				httptrace.Inject(ctx, req)

				fmt.Printf("Sending request to Service C ...\n")
				res, err := client.Do(req)
				if err != nil {
					panic(err)
				}
				body, err = ioutil.ReadAll(res.Body)
				_ = res.Body.Close()
				trace.SpanFromContext(ctx).SetStatus(codes.OK, "")

				return err
			})

		if err != nil {
			panic(err)
		}

		_, _ = io.WriteString(w, string(body))
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

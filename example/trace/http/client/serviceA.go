package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/codes"
)

func initTracer() {
	projectID := os.Getenv("PROJECT_ID")

	// Create Google Cloud Trace exporter to be able to retrieve
	// the collected spans.
	exporter, err := texporter.NewExporter(
		texporter.WithProjectID(projectID),
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

func main() {

	initTracer()
	tr := global.TraceProvider().Tracer("cloudtrace/example/client")

	client := http.DefaultClient
	ctx := correlation.NewContext(context.Background(),
		kv.String("username", "donuts"),
	)

	var body []byte

	scanner := bufio.NewScanner(os.Stdin)
	for true {
		fmt.Println("Enter your ingredient: \n (\"bye\" to quit)")
		scanner.Scan()
		ingredientName := scanner.Text()
		if ingredientName == "bye" {
			return
		}
		fmt.Println("Your text was: ", ingredientName)

		err := tr.WithSpan(ctx, "service A",
			func(ctx context.Context) error {
				// make sure the IP of service B
				req, _ := http.NewRequest("GET", "http://35.192.101.15:7777/"+ingredientName, nil)

				ctx, req = httptrace.W3C(ctx, req)
				httptrace.Inject(ctx, req)

				fmt.Printf("Sending request to Service B ...\n")
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

		fmt.Printf("Response Received:\n%s\n\n\n", body)
		fmt.Printf("Waiting for few seconds to export spans ...\n\n")
		fmt.Println("Check traces on Google Cloud Trace")
	}

}

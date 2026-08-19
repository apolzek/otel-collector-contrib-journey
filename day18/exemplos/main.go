// Programa que importa dois componentes de módulos vizinhos, ligados pelo
// go.work deste diretório, e imprime o que cada factory declara.
//
// Rode com: go run .
package main

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"

	"github.com/apolzek/otel-collector-contrib-journey/day07/printexporter"
	"github.com/apolzek/otel-collector-contrib-journey/day09/tagprocessor"
)

func main() {
	descreverExporter(printexporter.NewFactory())
	descreverProcessor(tagprocessor.NewFactory())
}

func descreverExporter(f exporter.Factory) {
	fmt.Println("exporter:", f.Type())
	fmt.Printf("  config padrão: %+v\n", f.CreateDefaultConfig())
	for _, sinal := range []pipeline.Signal{pipeline.SignalTraces, pipeline.SignalMetrics, pipeline.SignalLogs} {
		fmt.Printf("  %-8s %s\n", sinal, estabilidadeExporter(f, sinal))
	}
}

func estabilidadeExporter(f exporter.Factory, s pipeline.Signal) component.StabilityLevel {
	switch s {
	case pipeline.SignalTraces:
		return f.TracesStability()
	case pipeline.SignalMetrics:
		return f.MetricsStability()
	default:
		return f.LogsStability()
	}
}

func descreverProcessor(f processor.Factory) {
	fmt.Println("processor:", f.Type())
	fmt.Printf("  config padrão: %+v\n", f.CreateDefaultConfig())
	fmt.Println("  traces  ", f.TracesStability())
	fmt.Println("  logs    ", f.LogsStability())
}

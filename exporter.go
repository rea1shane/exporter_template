package exporter

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/rea1shane/gooooo/http"
	cases "github.com/rea1shane/gooooo/strings"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

// Exporter basic information.
type Exporter struct {
	Name           string // Name stylized as strings.SnakeCase, e.g. "node_exporter".
	Description    string // Description
	DefaultAddress string // DefaultAddress e.g. ":9100". Set "" to use env "PORT". (see gin.resolveAddress function)
}

// Run start server to collect metrics.
func (e Exporter) Run(logger *logrus.Logger) {
	var (
		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		disableExporterMetrics = kingpin.Flag(
			"web.disable-exporter-metrics",
			"Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).",
		).Bool()
		maxRequests = kingpin.Flag(
			"web.max-requests",
			"Maximum number of parallel scrape requests. Use 0 to disable.",
		).Default("40").Int()
		disableDefaultCollectors = kingpin.Flag(
			"collector.disable-defaults",
			"Set all collectors to disabled by default.",
		).Default("false").Bool()
		maxProcs = kingpin.Flag(
			"runtime.gomaxprocs",
			"The target number of CPUs Go will run on (GOMAXPROCS)",
		).Envar("GOMAXPROCS").Default("1").Int()
		address = kingpin.Flag(
			"web.listen-address",
			"Address on which to expose metrics and web interface. Not support multiple addresses.",
		).Default(e.DefaultAddress).String()

		logLevel = kingpin.Flag(
			"log.level",
			"Only log messages with the given severity or above. One of: [debug, info, warn, error]",
		).Default("info").String()
		latencyThreshold = kingpin.Flag(
			"web.latency_threshold",
			"When the latency exceeds the threshold, the log level will change from INFO to WARN. Use 0 to disable.",
		).Default("0").Duration()
	)
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(level)

	kingpin.Version(version.Print(e.Name))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *disableDefaultCollectors {
		DisableDefaultCollectors()
	}
	logger.Infof("Starting %s", e.Name)
	logger.Infof("Version: %s", version.Info())
	logger.Infof("Build context: %s", version.BuildContext())

	runtime.GOMAXPROCS(*maxProcs)
	logger.Debugf("Go MAXPROCS: %d", runtime.GOMAXPROCS(0))

	handler := http.NewHandler(logger, *latencyThreshold)
	handler.GET(*metricsPath, gin.WrapH(newHandler(!*disableExporterMetrics, *maxRequests, logger)))
	if *metricsPath != "/" {
		displayName, _ := cases.ConvertCase(e.Name, cases.PascalSnakeCase)
		landingConfig := web.LandingConfig{
			Name:        strings.ReplaceAll(displayName, "_", " "),
			Description: e.Description,
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Fatal(err)
		}
		handler.GET("/", gin.WrapH(landingPage))
	}

	switch *address {
	case "":
		handler.Run()
	default:
		handler.Run(*address)
	}
}

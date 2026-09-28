package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"web-crawler-go-test-proj/internal/config"
	"web-crawler-go-test-proj/internal/crawler"
	"web-crawler-go-test-proj/internal/exporter"
	"web-crawler-go-test-proj/internal/fetcher"
	"web-crawler-go-test-proj/internal/logger"
	"web-crawler-go-test-proj/internal/parser"
)

func main() {
	config := config.Load()
	logger, _, err := logger.InitLogger(config.LogFile)
	if err != nil {
		return
	}

	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	ctx, cancel := context.WithTimeout(
		ctx, config.Timeout,
	)
	defer cancel()

	parser := parser.NewHTMLParser()
	fetcher := fetcher.NewHTTPFetcher(config.ReqTimeout)

	crawler := crawler.NewCrawler(
		logger, parser, fetcher, config.URLs, config.ReqDepth, config.ReqTimeout,
	)

	exporter := exporter.NewExporter(logger)

	logger.Info("Starting crawler...")

	nodes := crawler.Run(ctx)

	if ctx.Err() != nil {
		logger.Warn("Crawler interrupted or timed out, exporting collected nodes...", "reason", ctx.Err())
	} else {
		logger.Info("Crawler finished successfully")
	}

	if err := exporter.ExportJSON(config.OutputFile, nodes); err != nil {
		logger.Error("Failed to export JSON", "error", err)
	}
}

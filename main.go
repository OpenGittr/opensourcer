package main

import (
	"github.com/opengittr/opensourcer/internal"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.NewCMD()

	// Initialize CLI service
	cliService := internal.NewService()

	// Catalog commands
	app.SubCommand("catalog", func(c *gofr.Context) (interface{}, error) {
		return cliService.ListCatalog(c)
	}, gofr.AddDescription("List available software in the catalog"))

	app.SubCommand("update", func(c *gofr.Context) (interface{}, error) {
		return cliService.Update(c)
	}, gofr.AddDescription("Update the local catalog from repository"))

	app.SubCommand("info", func(c *gofr.Context) (interface{}, error) {
		return cliService.GetInfo(c)
	}, gofr.AddDescription("Show details about a software"))

	// Deployment commands
	app.SubCommand("deploy", func(c *gofr.Context) (interface{}, error) {
		return cliService.Deploy(c)
	}, gofr.AddDescription("Deploy software locally or to cloud"))

	app.SubCommand("list", func(c *gofr.Context) (interface{}, error) {
		return cliService.List(c)
	}, gofr.AddDescription("List your deployments"))

	app.SubCommand("logs", func(c *gofr.Context) (interface{}, error) {
		return cliService.Logs(c)
	}, gofr.AddDescription("View logs for a deployment"))

	app.SubCommand("stop", func(c *gofr.Context) (interface{}, error) {
		return cliService.Stop(c)
	}, gofr.AddDescription("Stop a running deployment"))

	app.SubCommand("start", func(c *gofr.Context) (interface{}, error) {
		return cliService.Start(c)
	}, gofr.AddDescription("Start a stopped deployment"))

	app.SubCommand("destroy", func(c *gofr.Context) (interface{}, error) {
		return cliService.Destroy(c)
	}, gofr.AddDescription("Remove a deployment completely"))

	app.Run()
}
